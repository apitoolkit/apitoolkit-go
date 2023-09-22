package apitoolkit

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gorilla/mux"
)

// Middleware collects request, response parameters and publishes the payload
func (c *Client) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		errorList := []ATError{}
		newCtx := context.WithValue(req.Context(), ErrorListCtxKey, &errorList)
    req = req.WithContext(newCtx)

		reqBuf, _ := ioutil.ReadAll(req.Body)
		req.Body.Close()
		req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBuf))

		rec := httptest.NewRecorder()
		start := time.Now()
		next.ServeHTTP(rec, req)

		recRes := rec.Result()
		// io.Copy(res, recRes.Body)
		for k, v := range recRes.Header {
			for _, vv := range v {
				res.Header().Add(k, vv)
			}
		}
		resBody, _ := ioutil.ReadAll(recRes.Body)
		res.WriteHeader(recRes.StatusCode)
		res.Write(resBody)

		payload := c.buildPayload(GoDefaultSDKType, start, 
			req, recRes.StatusCode,
			reqBuf, resBody, recRes.Header, nil, req.URL.Path,
			c.config.RedactHeaders, c.config.RedactRequestBody, c.config.RedactResponseBody,
			errorList,
		)

		c.PublishMessage(req.Context(), payload)
	})
}


// Middleware collects request, response parameters and publishes the payload
func (c *Client) GorillaMuxMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		errorList := []ATError{}
		newCtx := context.WithValue(req.Context(), ErrorListCtxKey, &errorList)
    req = req.WithContext(newCtx)

		reqBuf, _ := ioutil.ReadAll(req.Body)
		req.Body.Close()
		req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBuf))

		rec := httptest.NewRecorder()
		start := time.Now()
		next.ServeHTTP(rec, req)

		recRes := rec.Result()
		for k, v := range recRes.Header {
			for _, vv := range v {
				res.Header().Add(k, vv)
			}
		}
		resBody, _ := ioutil.ReadAll(recRes.Body)
		res.WriteHeader(recRes.StatusCode)
		res.Write(resBody)

		route := mux.CurrentRoute(req)
		pathTmpl, _ := route.GetPathTemplate()
		vars := mux.Vars(req)

		payload := c.buildPayload(GoGorillaMux, start, 
			req, recRes.StatusCode,
			reqBuf, resBody, recRes.Header, vars, pathTmpl,
			c.config.RedactHeaders, c.config.RedactRequestBody, c.config.RedactResponseBody,
			errorList,
		)

		err := c.PublishMessage(req.Context(), payload)
		if err!=nil{
			if c.config.Debug {
				log.Println("APIToolkit: unable to publish request payload to pubsub.")
			}
		}
	})
}


