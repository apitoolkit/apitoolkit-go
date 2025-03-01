package apitoolkitchi

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	apt "github.com/apitoolkit/apitoolkit-go"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/honeycombio/otel-config-go/otelconfig"
	"go.opentelemetry.io/otel"
)

type Config struct {
	Debug               bool
	ServiceVersion      string
	ServiceName         string
	RedactHeaders       []string
	RedactRequestBody   []string
	RedactResponseBody  []string
	Tags                []string
	CaptureRequestBody  bool
	CaptureResponseBody bool
}

func ReportError(ctx context.Context, err error) {
	apt.ReportError(ctx, err)
}

func Middleware(config Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			tracer := otel.GetTracerProvider().Tracer(config.ServiceName)
			newCtx, span := tracer.Start(req.Context(), "apitoolkit-http-span")
			msgID := uuid.Must(uuid.NewRandom())
			newCtx = context.WithValue(newCtx, apt.CurrentRequestMessageID, msgID)
			errorList := []apt.ATError{}
			newCtx = context.WithValue(newCtx, apt.ErrorListCtxKey, &errorList)
			req = req.WithContext(newCtx)

			reqBuf, _ := io.ReadAll(req.Body)
			req.Body.Close()
			req.Body = io.NopCloser(bytes.NewBuffer(reqBuf))

			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, req)
			recRes := rec.Result()
			for k, v := range recRes.Header {
				for _, vv := range v {
					res.Header().Add(k, vv)
				}
			}
			resBody, _ := io.ReadAll(recRes.Body)
			res.WriteHeader(recRes.StatusCode)
			res.Write(resBody)

			aptConfig := apt.Config{
				ServiceName:         config.ServiceName,
				ServiceVersion:      config.ServiceVersion,
				Tags:                config.Tags,
				Debug:               config.Debug,
				CaptureRequestBody:  config.CaptureRequestBody,
				CaptureResponseBody: config.CaptureResponseBody,
				RedactHeaders:       config.RedactHeaders,
				RedactRequestBody:   config.RedactRequestBody,
				RedactResponseBody:  config.RedactResponseBody,
			}

			chiCtx := chi.RouteContext(req.Context())
			vars := map[string]string{}
			for i, key := range chiCtx.URLParams.Keys {
				if len(chiCtx.URLParams.Values) > i {
					vars[key] = chiCtx.URLParams.Values[i]
				}
			}

			payload := apt.BuildPayload(apt.GoGorillaMux,
				req, recRes.StatusCode,
				reqBuf, resBody, recRes.Header, vars, chiCtx.RoutePattern(),
				config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
				errorList,
				msgID,
				nil,
				aptConfig,
			)
			if config.Debug {
				log.Println(payload)
			}

			apt.CreateSpan(payload, aptConfig, span)

		})
	}
}

func ConfigureOpenTelemetry(opts ...otelconfig.Option) (func(), error) {
	opts = append([]otelconfig.Option{otelconfig.WithExporterEndpoint("otelcol.apitoolkit.io:4317"), otelconfig.WithExporterInsecure(true)}, opts...)
	return otelconfig.ConfigureOpenTelemetry(opts...)
}

var WithServiceName = otelconfig.WithServiceName
var WithServiceVersion = otelconfig.WithServiceVersion
var WithLogLevel = otelconfig.WithLogLevel
var WithResourceAttributes = otelconfig.WithResourceAttributes
var WithResourceOption = otelconfig.WithResourceOption
var WithPropagators = otelconfig.WithPropagators
var WithErrorHandler = otelconfig.WithErrorHandler
var WithMetricsReportingPeriod = otelconfig.WithMetricsReportingPeriod
var WithMetricsEnabled = otelconfig.WithMetricsEnabled
var WithTracesEnabled = otelconfig.WithTracesEnabled
var WithSpanProcessor = otelconfig.WithSpanProcessor
var WithSampler = otelconfig.WithSampler
