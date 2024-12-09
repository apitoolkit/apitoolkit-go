package apitoolkitnative

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

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
			msgID := uuid.Must(uuid.NewRandom())
			newCtx := context.WithValue(req.Context(), apt.CurrentRequestMessageID, msgID)

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
			apt.CreateSpan(payload, aptConfig)
		})
	}
}

func ConfigureOpenTelemetry(opts ...apt.Option) (func(), error) {
	return apt.ConfigureOpenTelemetry(opts...)
}

func WithServiceName(name string) apt.Option {
	return func(c *apt.OConfig) {
		c.ServiceName = name
	}
}
func WithServiceVersion(version string) apt.Option {
	return func(c *apt.OConfig) {
		c.ServiceVersion = version
	}
}

func WithLogLevel(loglevel string) apt.Option {
	return func(c *apt.OConfig) {
		c.LogLevel = loglevel
	}
}

func WithResourceAttributes(attributes map[string]string) apt.Option {
	return func(c *apt.OConfig) {
		for k, v := range attributes {
			c.ResourceAttributes[k] = v
		}
	}
}

func WithResourceOption(option resource.Option) apt.Option {
	return func(c *apt.OConfig) {
		c.ResourceOptions = append(c.ResourceOptions, option)
	}
}

func WithPropagators(propagators []string) apt.Option {
	return func(c *apt.OConfig) {
		c.Propagators = propagators
	}
}

// Configures a global error handler to be used throughout an OpenTelemetry instrumented project.
// See "go.opentelemetry.io/otel".
func WithErrorHandler(handler otel.ErrorHandler) apt.Option {
	return func(c *apt.OConfig) {
		c.ErrorHandler = handler
	}
}

func WithMetricsReportingPeriod(p time.Duration) apt.Option {
	return func(c *apt.OConfig) {
		c.MetricsReportingPeriod = fmt.Sprint(p)
	}
}

func WithMetricsEnabled(enabled bool) apt.Option {
	return func(c *apt.OConfig) {
		c.MetricsEnabled = &enabled
	}
}

func WithTracesEnabled(enabled bool) apt.Option {
	return func(c *apt.OConfig) {
		c.TracesEnabled = &enabled
	}
}

func WithSpanProcessor(sp ...trace.SpanProcessor) apt.Option {
	return func(c *apt.OConfig) {
		c.SpanProcessors = append(c.SpanProcessors, sp...)
	}
}

func WithSampler(sampler trace.Sampler) apt.Option {
	return func(c *apt.OConfig) {
		c.Sampler = sampler
	}
}
