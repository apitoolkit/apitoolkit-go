package apitoolkit

import (
	"bytes"
	"io/ioutil"
	"time"

	"github.com/gin-gonic/gin"
)

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

func (c *Client) GinMiddleware(ctx *gin.Context) {
	start := time.Now()
	reqByteBody, _ := ioutil.ReadAll(ctx.Request.Body)
	ctx.Request.Body = ioutil.NopCloser(bytes.NewBuffer(reqByteBody))

	blw := &ginBodyLogWriter{body: bytes.NewBuffer([]byte{}), ResponseWriter: ctx.Writer}
	ctx.Writer = blw

	ctx.Next()
	
	pathParams := map[string]string{}
	for _, param := range ctx.Params {
		pathParams[param.Key] = param.Value
	}

	payload := c.buildPayload(GoGinSDKType, start, 
		ctx.Request, ctx.Writer.Status(),
		reqByteBody, blw.body.Bytes(), ctx.Writer.Header().Clone(), 
		pathParams, ctx.FullPath(),
		c.config.RedactHeaders, c.config.RedactRequestBody, c.config.RedactResponseBody,
	)

	c.PublishMessage(ctx, payload)
}
