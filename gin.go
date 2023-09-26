package apitoolkit

import (
	"bytes"
	"context"
	"io/ioutil"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	msgID := uuid.Must(uuid.NewRandom())
	ctx.Set(string(CurrentRequestMessageID), msgID)
	errorList := []ATError{}
	ctx.Set(string(ErrorListCtxKey), &errorList)
	newCtx := context.WithValue(ctx.Request.Context(), ErrorListCtxKey, &errorList)
	newCtx = context.WithValue(newCtx, CurrentRequestMessageID, msgID)
	ctx.Request = ctx.Request.WithContext(newCtx)

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
		errorList,
		msgID,
		nil,
	)

	c.PublishMessage(ctx, payload)
}
