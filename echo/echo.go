package apitoolkitecho

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
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

func NewClient(ctx context.Context, conf apt.Config) (*apt.Client, error) {
	return apt.NewClient(ctx, conf)
}

func HTTPClient(ctx context.Context, opts ...apt.RoundTripperOption) *http.Client {
	return apt.HTTPClient(ctx, opts...)
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

func ReportError(ctx context.Context, err error) {
	apt.ReportError(ctx, err)
}

// EchoMiddleware middleware for echo framework, collects requests, response and publishes the payload
func EchoMiddleware(c *apt.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) (err error) {
			// Register the client in the context,
			// so it can be used for outgoing requests with little ceremony
			ctx.Set(string(apt.CurrentClient), c)

			msgID := uuid.Must(uuid.NewRandom())
			ctx.Set(string(apt.CurrentRequestMessageID), msgID)

			errorList := []apt.ATError{}
			ctx.Set(string(apt.ErrorListCtxKey), &errorList)
			newCtx := context.WithValue(ctx.Request().Context(), apt.ErrorListCtxKey, &errorList)
			newCtx = context.WithValue(newCtx, apt.CurrentRequestMessageID, msgID)
			newCtx = context.WithValue(newCtx, apt.CurrentClient, c)
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
			config := c.GetConfig()

			defer func() {
				if err := recover(); err != nil {
					if _, ok := err.(error); !ok {
						err = errors.New(err.(string))
					}
					apt.ReportError(ctx.Request().Context(), err.(error))
					payload := c.BuildPayload(apt.GoDefaultSDKType, startTime,
						ctx.Request(), 500,
						reqBuf, resBody.Bytes(), ctx.Response().Header().Clone(),
						pathParams, ctx.Path(),
						config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
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
			payload := c.BuildPayload(apt.GoDefaultSDKType, startTime,
				ctx.Request(), ctx.Response().Status,
				reqBuf, resBody.Bytes(), ctx.Response().Header().Clone(),
				pathParams, ctx.Path(),
				config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
				errorList,
				msgID,
				nil,
			)
			c.PublishMessage(ctx.Request().Context(), payload)
			return err
		}

	}
}
