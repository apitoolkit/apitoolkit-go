package apitoolkitgin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
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

type ginBodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *ginBodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *ginBodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func ReportError(ctx context.Context, err error) {
	apt.ReportError(ctx, err)
}

func Middleware(config Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Register the client in the context,
		// so it can be used for outgoing requests with little ceremony
		msgID := uuid.Must(uuid.NewRandom())
		ctx.Set(string(apt.CurrentRequestMessageID), msgID)
		errorList := []apt.ATError{}
		ctx.Set(string(apt.ErrorListCtxKey), &errorList)
		newCtx := context.WithValue(ctx.Request.Context(), apt.ErrorListCtxKey, &errorList)
		newCtx = context.WithValue(newCtx, apt.CurrentRequestMessageID, msgID)
		ctx.Request = ctx.Request.WithContext(newCtx)

		reqByteBody, _ := io.ReadAll(ctx.Request.Body)
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(reqByteBody))

		blw := &ginBodyLogWriter{body: bytes.NewBuffer([]byte{}), ResponseWriter: ctx.Writer}
		ctx.Writer = blw

		pathParams := map[string]string{}
		for _, param := range ctx.Params {
			pathParams[param.Key] = param.Value
		}
		aptConfig := getAptConfig(config)

		defer func() {
			if err := recover(); err != nil {
				if _, ok := err.(error); !ok {
					err = errors.New(err.(string))
				}
				apt.ReportError(ctx.Request.Context(), err.(error))
				payload := apt.BuildPayload(apt.GoGinSDKType,
					ctx.Request, 500,
					reqByteBody, blw.body.Bytes(), ctx.Writer.Header().Clone(),
					pathParams, ctx.FullPath(),
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
		ctx.Next()
		payload := apt.BuildPayload(apt.GoGinSDKType,
			ctx.Request, ctx.Writer.Status(),
			reqByteBody, blw.body.Bytes(), ctx.Writer.Header().Clone(),
			pathParams, ctx.FullPath(),
			config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
			errorList,
			msgID,
			nil,
			aptConfig,
		)
		if config.Debug {
			log.Println(payload)
		}
		apt.CreateSpan(payload, aptConfig)

	}
}

func getAptConfig(config Config) apt.Config {
	return apt.Config{
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
