package apitoolkitnative

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/google/uuid"
	"github.com/honeycombio/otel-config-go/otelconfig"
	"go.opentelemetry.io/otel"

	apt "github.com/apitoolkit/apitoolkit-go"
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

// Middleware collects request, response parameters and publishes the payload
func Middleware(config Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

			tracer := otel.GetTracerProvider().Tracer(config.ServiceName)
			newCtx, span := tracer.Start(req.Context(), "apnewCtxitoolkit-http-span")

			msgID := uuid.Must(uuid.NewRandom())
			newCtx = context.WithValue(newCtx, apt.CurrentRequestMessageID, msgID)

			errorList := []apt.ATError{}
			newCtx = context.WithValue(newCtx, apt.ErrorListCtxKey, &errorList)

			if config.ServiceName == "" {
				config.ServiceName = os.Getenv("OTEL_SERVICE_NAME")
			}

			req = req.WithContext(newCtx)

			reqBuf, _ := io.ReadAll(req.Body)
			req.Body.Close()
			req.Body = io.NopCloser(bytes.NewBuffer(reqBuf))

			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, req)

			recRes := rec.Result()
			// io.Copy(res, recRes.Body)
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

			payload := apt.BuildPayload(apt.GoDefaultSDKType,
				req, recRes.StatusCode,
				reqBuf, resBody, recRes.Header, nil, req.URL.Path,
				config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
				errorList,
				msgID,
				nil,
				aptConfig,
			)
			if config.Debug {
				log.Printf("payload: %+v\n", payload)
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
