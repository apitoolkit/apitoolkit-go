package apitoolkitfiber

import (
	"context"
	"errors"
	"fmt"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
	fiber "github.com/gofiber/fiber/v2"
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

func Middleware(config Config) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// Register the client in the context,
		// so it can be used for outgoing requests with little ceremony

		msgID := uuid.Must(uuid.NewRandom())
		ctx.Locals(string(apt.CurrentRequestMessageID), msgID)
		errorList := []apt.ATError{}
		ctx.Locals(string(apt.ErrorListCtxKey), &errorList)
		newCtx := context.WithValue(ctx.Context(), apt.ErrorListCtxKey, &errorList)
		newCtx = context.WithValue(newCtx, apt.CurrentRequestMessageID, msgID)
		ctx.SetUserContext(newCtx)
		respHeaders := map[string][]string{}
		for k, v := range ctx.GetRespHeaders() {
			respHeaders[k] = v
		}
		aptConfig := getAptConfig(config)
		defer func() {
			if err := recover(); err != nil {
				if _, ok := err.(error); !ok {
					err = errors.New(err.(string))
				}
				apt.ReportError(ctx.UserContext(), err.(error))
				payload := apt.BuildFastHTTPPayload(apt.GoFiberSDKType,
					ctx.Context(), 500,
					ctx.Request().Body(), ctx.Response().Body(), respHeaders,
					ctx.AllParams(), ctx.Route().Path,
					config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
					errorList,
					msgID,
					nil,
					string(ctx.Context().Referer()),
					aptConfig,
				)
				apt.CreateSpan(payload, aptConfig)
				panic(err)
			}
		}()

		err := ctx.Next()
		payload := apt.BuildFastHTTPPayload(apt.GoFiberSDKType,
			ctx.Context(), ctx.Response().StatusCode(),
			ctx.Request().Body(), ctx.Response().Body(), respHeaders,
			ctx.AllParams(), ctx.Route().Path,
			config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
			errorList,
			msgID,
			nil,
			string(ctx.Context().Referer()),
			aptConfig,
		)

		apt.CreateSpan(payload, aptConfig)
		return err
	}
}

func ReportError(ctx context.Context, err error) {
	apt.ReportError(ctx, err)
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
