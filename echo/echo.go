package apitoolkitecho

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

// bodyDumpResponseWriter use to preserve the http response body during request processing
type echoBodyLogWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *echoBodyLogWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *echoBodyLogWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *echoBodyLogWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *echoBodyLogWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

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

// EchoMiddleware middleware for echo framework, collects requests, response and publishes the payload
func Middleware(config Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) (err error) {
			// Register the client in the context,
			// so it can be used for outgoing requests with little ceremony

			msgID := uuid.Must(uuid.NewRandom())
			ctx.Set(string(apt.CurrentRequestMessageID), msgID)

			errorList := []apt.ATError{}
			ctx.Set(string(apt.ErrorListCtxKey), &errorList)
			newCtx := context.WithValue(ctx.Request().Context(), apt.ErrorListCtxKey, &errorList)
			newCtx = context.WithValue(newCtx, apt.CurrentRequestMessageID, msgID)
			ctx.SetRequest(ctx.Request().WithContext(newCtx))

			var reqBuf []byte
			// safely read request body
			if ctx.Request().Body != nil {
				reqBuf, _ = io.ReadAll(ctx.Request().Body)
			}
			ctx.Request().Body = io.NopCloser(bytes.NewBuffer(reqBuf))
			// create a MultiWriter that streams the response body into resBody
			resBody := new(bytes.Buffer)
			mw := io.MultiWriter(ctx.Response().Writer, resBody)
			writer := &echoBodyLogWriter{Writer: mw, ResponseWriter: ctx.Response().Writer}
			ctx.Response().Writer = writer
			pathParams := map[string]string{}
			for _, paramName := range ctx.ParamNames() {
				pathParams[paramName] = ctx.Param(paramName)
			}
			aptConfig := apt.Config{
				ServiceName:         config.ServiceName,
				ServiceVersion:      config.ServiceVersion,
				Tags:                config.Tags,
				CaptureRequestBody:  config.CaptureRequestBody,
				CaptureResponseBody: config.CaptureResponseBody,
				RedactHeaders:       config.RedactHeaders,
				RedactRequestBody:   config.RedactRequestBody,
				RedactResponseBody:  config.RedactResponseBody,
			}

			defer func() {
				if err := recover(); err != nil {
					if _, ok := err.(error); !ok {
						err = errors.New(err.(string))
					}
					apt.ReportError(ctx.Request().Context(), err.(error))
					payload := apt.BuildPayload(apt.GoDefaultSDKType,
						ctx.Request(), 500,
						reqBuf, resBody.Bytes(), ctx.Response().Header().Clone(),
						pathParams, ctx.Path(),
						config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
						errorList,
						msgID,
						nil,
						aptConfig,
					)
					apt.CreateSpan(payload, aptConfig)
					panic(err)
				}
			}()

			// pass on request handling
			err = next(ctx)

			// proceed post-response processing
			payload := apt.BuildPayload(apt.GoDefaultSDKType,
				ctx.Request(), ctx.Response().Status,
				reqBuf, resBody.Bytes(), ctx.Response().Header().Clone(),
				pathParams, ctx.Path(),
				config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
				errorList,
				msgID,
				nil,
				aptConfig,
			)
			apt.CreateSpan(payload, aptConfig)
			return err
		}
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
