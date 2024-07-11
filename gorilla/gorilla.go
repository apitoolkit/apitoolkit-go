package apitoolkitgorilla

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

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

// GorillaMuxMiddleware is for the gorilla mux routing library and collects request, response parameters and publishes the payload
func GorillaMuxMiddleware(c *apt.Client) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			msgID := uuid.Must(uuid.NewRandom())
			newCtx := context.WithValue(req.Context(), apt.CurrentRequestMessageID, msgID)

			errorList := []apt.ATError{}
			newCtx = context.WithValue(newCtx, apt.ErrorListCtxKey, &errorList)
			newCtx = context.WithValue(newCtx, apt.CurrentClient, c)
			req = req.WithContext(newCtx)

			reqBuf, _ := io.ReadAll(req.Body)
			req.Body.Close()
			req.Body = io.NopCloser(bytes.NewBuffer(reqBuf))

			rec := httptest.NewRecorder()
			start := time.Now()
			next.ServeHTTP(rec, req)

			recRes := rec.Result()
			for k, v := range recRes.Header {
				for _, vv := range v {
					res.Header().Add(k, vv)
				}
			}
			resBody, _ := io.ReadAll(recRes.Body)
			res.WriteHeader(recRes.StatusCode)
			res.Write(resBody)

			route := mux.CurrentRoute(req)
			pathTmpl, _ := route.GetPathTemplate()
			vars := mux.Vars(req)
			config := c.GetConfig()

			payload := c.BuildPayload(apt.GoGorillaMux, start,
				req, recRes.StatusCode,
				reqBuf, resBody, recRes.Header, vars, pathTmpl,
				config.RedactHeaders, config.RedactRequestBody, config.RedactResponseBody,
				errorList,
				msgID,
				nil,
			)

			err := c.PublishMessage(req.Context(), payload)
			if err != nil {
				if config.Debug {
					log.Println("APIToolkit: unable to publish request payload to pubsub.")
				}
			}
		})
	}
}
