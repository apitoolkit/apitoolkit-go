package apitoolkitnative

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
	"github.com/cockroachdb/errors"
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
	h := Middleware(client)
	ts := httptest.NewServer(h(http.HandlerFunc(handlerFn)))
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

func TestReporting(t *testing.T) {
	ctx := context.Background()
	cfg := apt.Config{
		APIKey:             os.Getenv("APITOOLKIT_KEY"),
		RootURL:            "",
		RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
		RedactResponseBody: apt.ExampleDataRedaction,
		Tags:               []string{"staging"},
	}
	client, err := NewClient(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	assert.NoError(t, err)

	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		err1 := errors.Newf("Example Error %v", "value")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"key":"value"}`))
		err2 := errors.Wrap(err1, "wrapper from err2")
		ReportError(r.Context(), err2)
	}

	ts := httptest.NewServer(Middleware(client)(http.HandlerFunc(handlerFn)))
	defer ts.Close()

	atHTTPClient := http.DefaultClient
	atHTTPClient.Transport = client.WrapRoundTripper(
		ctx, atHTTPClient.Transport,
		WithRedactHeaders("ABC"),
	)
	req.SetClient(atHTTPClient)
	_, err = req.Post(ts.URL+"/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(apt.ExampleData2),
	)

	assert.NoError(t, err)
}

func TestSugaredReporting(t *testing.T) {
	ctx := context.Background()
	cfg := apt.Config{
		APIKey:             os.Getenv("APITOOLKIT_KEY"),
		RootURL:            "",
		RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
		RedactResponseBody: apt.ExampleDataRedaction,
		Tags:               []string{"staging"},
	}
	client, err := NewClient(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	assert.NoError(t, err)

	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		err1 := errors.Newf("Example Error %v", "value")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"key":"value"}`))
		err2 := errors.Wrap(err1, "wrapper from err2")
		ReportError(r.Context(), err2)
	}

	ts := httptest.NewServer(Middleware(client)(http.HandlerFunc(handlerFn)))
	defer ts.Close()

	_, err = req.Post(ts.URL+"/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(apt.ExampleData2),
	)

	assert.NoError(t, err)
}

func TestOutgoingMiddleware(t *testing.T) {
	client := &apt.Client{}
	client.SetConfig(&apt.Config{
		RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
		RedactResponseBody: apt.ExampleDataRedaction,
	})
	var publishCalled bool
	client.PublishMessage = func(ctx context.Context, payload apt.Payload) error {
		if payload.URLPath == "/test" {
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

		} else {
			assert.Equal(t, "GET", payload.Method)
			assert.Equal(t, "/from-gorilla", payload.URLPath)
			assert.Equal(t, map[string]string(nil), payload.PathParams)
			assert.Equal(t, map[string][]string{
				"param1": {"abc"},
				"param2": {"123"},
			}, payload.QueryParams)

			assert.Equal(t, "/from-gorilla?param1=abc&param2=123", payload.RawURL)
			assert.Equal(t, http.StatusServiceUnavailable, payload.StatusCode)
			assert.Greater(t, payload.Duration, 1000*time.Nanosecond)
			assert.Equal(t, apt.GoOutgoing, payload.SdkType)
			assert.NotNil(t, payload.ParentID)

		}
		return nil
	}

	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NotEmpty(t, body)
		atHTTPClient := HTTPClient(r.Context())
		_, _ = atHTTPClient.Get("http://localhost:3000/from-gorilla?param1=abc&param2=123")
		jsonByte, err := json.Marshal(apt.ExampleData)
		assert.NoError(t, err)

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("X-API-KEY", "applicationKey")
		w.WriteHeader(http.StatusAccepted)
		w.Write(jsonByte)
	}

	ts := httptest.NewServer(Middleware(client)(http.HandlerFunc(handlerFn)))

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

func TestErrorReporting(t *testing.T) {
	client := &apt.Client{}
	client.SetConfig(&apt.Config{
		RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
		RedactResponseBody: apt.ExampleDataRedaction,
	})

	var publishCalled bool
	client.PublishMessage = func(ctx context.Context, payload apt.Payload) error {
		// x, _ := json.MarshalIndent(payload, "", "\t")
		// fmt.Println(string(x))
		assert.NotEmpty(t, payload.Errors)
		assert.Equal(t, "wrapper from err2 Example Error value", payload.Errors[0].Message)
		assert.Equal(t, "Example Error value", payload.Errors[0].RootErrorMessage)
		assert.Equal(t, "*fmt.wrapError", payload.Errors[0].ErrorType)
		assert.Equal(t, "*errors.errorString", payload.Errors[0].RootErrorType)

		assert.Equal(t, "POST", payload.Method)
		assert.Equal(t, "/test", payload.URLPath)
		publishCalled = true
		return nil
	}

	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		err1 := fmt.Errorf("Example Error %v", "value")

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"key":"value"}`))

		err2 := fmt.Errorf("wrapper from err2 %w", err1)
		ReportError(r.Context(), err2)
	}

	ts := httptest.NewServer(Middleware(client)(http.HandlerFunc(handlerFn)))
	defer ts.Close()

	outClient := &apt.Client{}
	outClient.SetConfig(&apt.Config{})

	outClient.PublishMessage = func(ctx context.Context, payload apt.Payload) error {
		assert.Equal(t, "/test?param1=abc&param2=123", payload.RawURL)
		assert.Equal(t, http.StatusAccepted, payload.StatusCode)
		assert.Greater(t, payload.Duration, 1000*time.Nanosecond)
		assert.Equal(t, apt.GoOutgoing, payload.SdkType)
		return nil
	}

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

func TestReportingInteg(t *testing.T) {
	ctx := context.Background()
	cfg := apt.Config{
		APIKey:             os.Getenv("APITOOLKIT_KEY"),
		RootURL:            "",
		RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
		RedactResponseBody: apt.ExampleDataRedaction,
		Tags:               []string{"staging"},
	}
	client, err := NewClient(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	assert.NoError(t, err)

	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		err1 := errors.Newf("Example Error %v", "value")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"key":"value"}`))
		err2 := errors.Wrap(err1, "wrapper from err2")
		ReportError(r.Context(), err2)
	}
	ts := httptest.NewServer(Middleware(client)(http.HandlerFunc(handlerFn)))
	defer ts.Close()

	_, err = req.Post(ts.URL+"/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(apt.ExampleData2),
	)

	assert.NoError(t, err)
}
