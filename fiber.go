package apitoolkit

import (
	"context"
	"errors"
	"time"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func (c *Client) FiberMiddleware(ctx *fiber.Ctx) error {
	// Register the client in the context,
	// so it can be used for outgoing requests with little ceremony
	ctx.Locals(string(CurrentClient), c)

	msgID := uuid.Must(uuid.NewRandom())
	ctx.Locals(string(CurrentRequestMessageID), msgID)
	errorList := []ATError{}
	ctx.Locals(string(ErrorListCtxKey), &errorList)
	ctx.Locals(CurrentClient, c)
	newCtx := context.WithValue(ctx.Context(), ErrorListCtxKey, &errorList)
	newCtx = context.WithValue(newCtx, CurrentClient, c)
	newCtx = context.WithValue(newCtx, CurrentRequestMessageID, msgID)
	ctx.SetUserContext(newCtx)
	respHeaders := map[string][]string{}
	for k, v := range ctx.GetRespHeaders() {
		respHeaders[k] = v
	}

	start := time.Now()
	defer func() {
		if err := recover(); err != nil {
			if _, ok := err.(error); !ok {
				err = errors.New(err.(string))
			}
			ReportError(ctx.UserContext(), err.(error))
			payload := c.buildFastHTTPPayload(GoFiberSDKType, start,
				ctx.Context(), 500,
				ctx.Request().Body(), ctx.Response().Body(), respHeaders,
				ctx.AllParams(), ctx.Route().Path,
				c.config.RedactHeaders, c.config.RedactRequestBody, c.config.RedactResponseBody,
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
	payload := c.buildFastHTTPPayload(GoFiberSDKType, start,
		ctx.Context(), ctx.Response().StatusCode(),
		ctx.Request().Body(), ctx.Response().Body(), respHeaders,
		ctx.AllParams(), ctx.Route().Path,
		c.config.RedactHeaders, c.config.RedactRequestBody, c.config.RedactResponseBody,
		errorList,
		msgID,
		nil,
		string(ctx.Context().Referer()),
	)

	c.PublishMessage(ctx.Context(), payload)
	return err
}
