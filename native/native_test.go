package apitoolkitnative

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
	"github.com/imroc/req"
	"github.com/stretchr/testify/assert"
)

func TestNativeGoMiddleware(t *testing.T) {
	client := &apt.Client{}
	client.SetConfig(&apt.Config{
		RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
		RedactResponseBody: apt.ExampleDataRedaction,
	})
	var publishCalled bool
	client.PublishMessage = func(ctx context.Context, payload apt.Payload) error {
		assert.Equal(t, "POST", payload.Method)
		assert.Equal(t, "/test", payload.URLPath)
		assert.Equal(t, map[string]string(nil), payload.PathParams)
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
		assert.Equal(t, "/test?param1=abc&param2=123", payload.RawURL)
		assert.Equal(t, http.StatusAccepted, payload.StatusCode)
		assert.Greater(t, payload.Duration, 1000*time.Nanosecond)
		assert.Equal(t, apt.GoDefaultSDKType, payload.SdkType)

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

	ts := httptest.NewServer(client.Middleware(http.HandlerFunc(handlerFn)))
	defer ts.Close()

	_, err := req.Post(ts.URL+"/test",
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
