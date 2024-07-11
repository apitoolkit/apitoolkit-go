package apitoolkitgin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
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

type Config struct {
	Debug              bool
	VerboseDebug       bool
	RootURL            string
	APIKey             string
	ProjectID          string
	ServiceVersion     string
	RedactHeaders      []string
	RedactRequestBody  []string
	RedactResponseBody []string
	Tags               []string `json:"tags"`
}

func NewClient(ctx context.Context, conf Config) (*apt.Client, error) {
	config := apt.Config{
		Debug:              conf.Debug,
		VerboseDebug:       conf.VerboseDebug,
		RootURL:            conf.RootURL,
		APIKey:             conf.APIKey,
		ProjectID:          conf.ProjectID,
		ServiceVersion:     conf.ServiceVersion,
		RedactHeaders:      conf.RedactHeaders,
		RedactRequestBody:  conf.RedactRequestBody,
		RedactResponseBody: conf.RedactResponseBody,
		Tags:               conf.Tags,
	}
	return apt.NewClient(ctx, config)
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

func GinMiddleware(c *apt.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Register the client in the context,
		// so it can be used for outgoing requests with little ceremony
		ctx.Set(string(apt.CurrentClient), c)

		msgID := uuid.Must(uuid.NewRandom())
		ctx.Set(string(apt.CurrentRequestMessageID), msgID)
		errorList := []apt.ATError{}
		ctx.Set(string(apt.ErrorListCtxKey), &errorList)
		newCtx := context.WithValue(ctx.Request.Context(), apt.ErrorListCtxKey, &errorList)
		newCtx = context.WithValue(newCtx, apt.CurrentClient, c)
		newCtx = context.WithValue(newCtx, apt.CurrentRequestMessageID, msgID)
		ctx.Request = ctx.Request.WithContext(newCtx)

		start := time.Now()
		reqByteBody, _ := io.ReadAll(ctx.Request.Body)
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(reqByteBody))

		blw := &ginBodyLogWriter{body: bytes.NewBuffer([]byte{}), ResponseWriter: ctx.Writer}
		ctx.Writer = blw

		pathParams := map[string]string{}
		for _, param := range ctx.Params {
			pathParams[param.Key] = param.Value
		}
		config := c.GetConfig()

		defer func() {
			if err := recover(); err != nil {
				if _, ok := err.(error); !ok {
					err = errors.New(err.(string))
				}
				apt.ReportError(ctx.Request.Context(), err.(error))
				payload := c.BuildPayload(apt.GoGinSDKType, start,
					ctx.Request, 500,
					reqByteBody, blw.body.Bytes(), ctx.Writer.Header().Clone(),
					pathParams, ctx.FullPath(),
					config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
					errorList,
					msgID,
					nil,
				)
				c.PublishMessage(ctx, payload)
				panic(err)
			}
		}()

		ctx.Next()

		payload := c.BuildPayload(apt.GoGinSDKType, start,
			ctx.Request, ctx.Writer.Status(),
			reqByteBody, blw.body.Bytes(), ctx.Writer.Header().Clone(),
			pathParams, ctx.FullPath(),
			config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
			errorList,
			msgID,
			nil,
		)

		c.PublishMessage(ctx, payload)

	}
}
