package apitoolkit

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GorillaMuxMiddleware is for the gorilla mux routing library and collects request, response parameters and publishes the payload
func (c *Client) GorillaMuxMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		msgID := uuid.Must(uuid.NewRandom())
		newCtx := context.WithValue(req.Context(), CurrentRequestMessageID, msgID)

		errorList := []ATError{}
		newCtx = context.WithValue(newCtx, ErrorListCtxKey, &errorList)
		newCtx = context.WithValue(newCtx, CurrentClient, c)
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

		payload := c.BuildPayload(GoGorillaMux, start,
			req, recRes.StatusCode,
			reqBuf, resBody, recRes.Header, vars, pathTmpl,
			c.config.RedactHeaders, c.config.RedactRequestBody, c.config.RedactResponseBody,
			errorList,
			msgID,
			nil,
		)

		err := c.PublishMessage(req.Context(), payload)
		if err != nil {
			if c.config.Debug {
				log.Println("APIToolkit: unable to publish request payload to pubsub.")
			}
		}
	})
}
