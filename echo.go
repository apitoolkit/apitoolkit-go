package apitoolkit

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"time"

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

		// pass on request handling
		if err = next(ctx); err != nil {
			ctx.Error(err)
		}

		pathParams := map[string]string{}
		for _, paramName := range ctx.ParamNames() {
			pathParams[paramName] =  ctx.Param(paramName)
		}

		// proceed post-response processing
		payload := c.buildPayload(GoDefaultSDKType, startTime, 
			ctx.Request(), ctx.Response().Status,
			reqBuf, resBody.Bytes(), ctx.Response().Header().Clone(),
			pathParams, ctx.Path(),
			c.config.RedactHeaders, c.config.RedactRequestBody, c.config.RedactResponseBody,
		)
		c.PublishMessage(ctx.Request().Context(), payload)
		return
	}
}

