package apitoolkit

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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

// EchoMiddleware middleware for echo framework, collects requests, response and publishes the payload
func (c *Client) EchoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) (err error) {
		// Register the client in the context,
		// so it can be used for outgoing requests with little ceremony
		ctx.Set(string(CurrentClient), c)

		msgID := uuid.Must(uuid.NewRandom())
		ctx.Set(string(CurrentRequestMessageID), msgID)

		errorList := []ATError{}
		ctx.Set(string(ErrorListCtxKey), &errorList)
		newCtx := context.WithValue(ctx.Request().Context(), ErrorListCtxKey, &errorList)
		newCtx = context.WithValue(newCtx, CurrentRequestMessageID, msgID)
		newCtx = context.WithValue(newCtx, CurrentClient, c)
		ctx.SetRequest(ctx.Request().WithContext(newCtx))

		var reqBuf []byte
		// safely read request body
		if ctx.Request().Body != nil {
			reqBuf, _ = io.ReadAll(ctx.Request().Body)
		}
		ctx.Request().Body = io.NopCloser(bytes.NewBuffer(reqBuf))
		startTime := time.Now()

		// create a MultiWriter that streams the response body into resBody
		resBody := new(bytes.Buffer)
		mw := io.MultiWriter(ctx.Response().Writer, resBody)
		writer := &echoBodyLogWriter{Writer: mw, ResponseWriter: ctx.Response().Writer}
		ctx.Response().Writer = writer
		pathParams := map[string]string{}
		for _, paramName := range ctx.ParamNames() {
			pathParams[paramName] = ctx.Param(paramName)
		}

		defer func() {
			if err := recover(); err != nil {
				ReportError(ctx.Request().Context(), err.(error))
				payload := c.buildPayload(GoDefaultSDKType, startTime,
					ctx.Request(), 500,
					reqBuf, resBody.Bytes(), ctx.Response().Header().Clone(),
					pathParams, ctx.Path(),
					c.config.RedactHeaders, c.config.RedactRequestBody, c.config.RedactResponseBody,
					errorList,
					msgID,
					nil,
				)
				c.PublishMessage(ctx.Request().Context(), payload)
				panic(err)
			}
		}()

		// pass on request handling
		err = next(ctx)

		// proceed post-response processing
		payload := c.buildPayload(GoDefaultSDKType, startTime,
			ctx.Request(), ctx.Response().Status,
			reqBuf, resBody.Bytes(), ctx.Response().Header().Clone(),
			pathParams, ctx.Path(),
			c.config.RedactHeaders, c.config.RedactRequestBody, c.config.RedactResponseBody,
			errorList,
			msgID,
			nil,
		)
		c.PublishMessage(ctx.Request().Context(), payload)
		return err
	}
}
