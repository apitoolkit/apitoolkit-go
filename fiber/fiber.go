package apitoolkitfiber

import (
	"context"
	"errors"
	"net/http"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
	fiber "github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func NewClient(ctx context.Context, conf apt.Config) (*apt.Client, error) {
	return apt.NewClient(ctx, conf)
}

func HTTPClient(ctx context.Context, opts ...apt.RoundTripperOption) *http.Client {
	return apt.HTTPClient(ctx, opts...)
}

func ReportError(ctx context.Context, err error) {
	apt.ReportError(ctx, err)
}

func WithRedactHeaders(headers ...string) apt.RoundTripperOption {
	return apt.WithRedactHeaders(headers...)
}
func WithRedactRequestBody(paths ...string) apt.RoundTripperOption {
	return apt.WithRedactRequestBody(paths...)
}
func WithRedactResponseBody(paths ...string) apt.RoundTripperOption {
	return apt.WithRedactResponseBody(paths...)
}

func FiberMiddleware(c *apt.Client) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// Register the client in the context,
		// so it can be used for outgoing requests with little ceremony
		ctx.Locals(string(apt.CurrentClient), c)

		msgID := uuid.Must(uuid.NewRandom())
		ctx.Locals(string(apt.CurrentRequestMessageID), msgID)
		errorList := []apt.ATError{}
		ctx.Locals(string(apt.ErrorListCtxKey), &errorList)
		ctx.Locals(apt.CurrentClient, c)
		newCtx := context.WithValue(ctx.Context(), apt.ErrorListCtxKey, &errorList)
		newCtx = context.WithValue(newCtx, apt.CurrentClient, c)
		newCtx = context.WithValue(newCtx, apt.CurrentRequestMessageID, msgID)
		ctx.SetUserContext(newCtx)
		respHeaders := map[string][]string{}
		for k, v := range ctx.GetRespHeaders() {
			respHeaders[k] = v
		}
		start := time.Now()
		config := c.GetConfig()
		defer func() {
			if err := recover(); err != nil {
				if _, ok := err.(error); !ok {
					err = errors.New(err.(string))
				}
				apt.ReportError(ctx.UserContext(), err.(error))
				payload := c.BuildFastHTTPPayload(apt.GoFiberSDKType, start,
					ctx.Context(), 500,
					ctx.Request().Body(), ctx.Response().Body(), respHeaders,
					ctx.AllParams(), ctx.Route().Path,
					config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
					errorList,
					msgID,
					nil,
					string(ctx.Context().Referer()),
				)
				c.PublishMessage(ctx.Context(), payload)
				panic(err)
			}
		}()

		err := ctx.Next()
		payload := c.BuildFastHTTPPayload(apt.GoFiberSDKType, start,
			ctx.Context(), ctx.Response().StatusCode(),
			ctx.Request().Body(), ctx.Response().Body(), respHeaders,
			ctx.AllParams(), ctx.Route().Path,
			config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
			errorList,
			msgID,
			nil,
			string(ctx.Context().Referer()),
		)

		c.PublishMessage(ctx.Context(), payload)
		return err
	}
}
