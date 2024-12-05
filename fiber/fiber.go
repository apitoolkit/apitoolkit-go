package apitoolkitfiber

import (
	"context"
	"errors"

	apt "github.com/apitoolkit/apitoolkit-go"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"go.opentelemetry.io/otel/trace"
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
	Tracer              trace.Tracer
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
		Tracer:              config.Tracer,
	}
}

func Middleware(config Config) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// Register the client in the context,
		// so it can be used for outgoing requests with little ceremony
		_, span := config.Tracer.Start(ctx.Context(), string(apt.SpanName))
		ctx.Locals(string(apt.CurrentSpan), span)

		msgID := uuid.Must(uuid.NewRandom())
		ctx.Locals(string(apt.CurrentRequestMessageID), msgID)
		errorList := []apt.ATError{}
		ctx.Locals(string(apt.ErrorListCtxKey), &errorList)
		ctx.Locals(apt.CurrentSpan, span)
		newCtx := context.WithValue(ctx.Context(), apt.ErrorListCtxKey, &errorList)
		newCtx = context.WithValue(newCtx, apt.CurrentSpan, span)
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
				apt.CreateSpan(payload, aptConfig, span)
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

		apt.CreateSpan(payload, aptConfig, span)
		return err
	}
}

func ReportError(ctx context.Context, err error) {
	apt.ReportError(ctx, err)
}
