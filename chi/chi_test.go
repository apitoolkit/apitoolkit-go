package apitoolkitchi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
	"github.com/go-chi/chi/v5"
	"github.com/imroc/req"
	"github.com/stretchr/testify/assert"
)

func TestChiMiddleware(t *testing.T) {
	client := &apt.Client{}
	client.SetConfig(&apt.Config{
		Debug:              true,
		VerboseDebug:       true,
		RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
		RedactResponseBody: apt.ExampleDataRedaction,
	})
	var publishCalled bool
	client.PublishMessage = func(ctx context.Context, payload apt.Payload) error {
		assert.Equal(t, "POST", payload.Method)
		assert.Equal(t, "/{param1:[a-z]+}/test", payload.URLPath)
		assert.Equal(t, map[string]string{"param1": "paramval"}, payload.PathParams)
		assert.Equal(t, map[string][]string{
			"param1": {"abc"},
			"param2": {"123"},
		}, payload.QueryParams)

		assert.Equal(t, map[string][]string{
			"Accept-Encoding": {"gzip"},
			"Content-Length":  {"437"},
			"Content-Type":    {"application/json"},
			"User-Agent":      {"Go-http-client/1.1"},
			"X-Api-Key":       {"past-3"},
		}, payload.RequestHeaders)
		assert.Equal(t, map[string][]string{
			"Content-Type": {"application/json"},
			"X-Api-Key":    {"applicationKey"},
		}, payload.ResponseHeaders)
		assert.Equal(t, "/paramval/test?param1=abc&param2=123", payload.RawURL)
		assert.Equal(t, http.StatusAccepted, payload.StatusCode)
		assert.Greater(t, payload.Duration, 1000*time.Nanosecond)
		assert.Equal(t, apt.GoGorillaMux, payload.SdkType)

		reqData, _ := json.Marshal(apt.ExampleData2)
		respData, _ := json.Marshal(apt.ExampleDataRedacted)

		assert.Equal(t, reqData, payload.RequestBody)
		assert.Equal(t, respData, payload.ResponseBody)

		publishCalled = true
		return nil
	}

	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NotEmpty(t, body)

		jsonByte, err := json.Marshal(apt.ExampleData)
		assert.NoError(t, err)

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("X-API-KEY", "applicationKey")
		w.WriteHeader(http.StatusAccepted)
		w.Write(jsonByte)
	}

	r := chi.NewRouter()
	r.Use(ChiMiddleware(client))
	r.Post("/{param1:[a-z]+}/test", handlerFn)

	ts := httptest.NewServer(r)
	defer ts.Close()

	_, err := req.Post(ts.URL+"/paramval/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(apt.ExampleData2),
	)
	assert.NoError(t, err)
	assert.True(t, publishCalled)
}

func TestOutgoingRequestChi(t *testing.T) {
	client := &apt.Client{}
	client.SetConfig(&apt.Config{})
	var publishCalled bool
	router := chi.NewRouter()
	router.Use((ChiMiddleware(client)))
	var parentId *string
	client.PublishMessage = func(ctx context.Context, payload apt.Payload) error {
		if payload.RawURL == "/from-gorilla" {
			assert.NotNil(t, payload.ParentID)
			parentId = payload.ParentID
		} else if payload.URLPath == "/:slug/test" {
			assert.Equal(t, *parentId, payload.MsgID)
		}
		publishCalled = true
		return nil
	}
	router.Get("/:slug/test", func(w http.ResponseWriter, r *http.Request) {
		hClient := HTTPClient(r.Context())
		_, _ = hClient.Get("http://localhost:3000/from-gorilla")

		fmt.Fprint(w, "Hello world")
	})

	ts := httptest.NewServer(router)
	defer ts.Close()
	_, err := req.Get(ts.URL + "/slug-value/test")
	assert.NoError(t, err)
	assert.True(t, publishCalled)
}
