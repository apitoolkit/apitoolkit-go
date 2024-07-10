package apitoolkitgin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"github.com/stretchr/testify/assert"
)

func TestGinMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	client := &apt.Client{}
	client.SetConfig(&apt.Config{
		RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
		RedactResponseBody: apt.ExampleDataRedaction,
	})

	var publishCalled bool

	client.PublishMessage = func(ctx context.Context, payload apt.Payload) error {
		assert.Equal(t, "POST", payload.Method)
		assert.Equal(t, "/:slug/test", payload.URLPath)
		assert.Equal(t, map[string]string{
			"slug": "slug-value",
		}, payload.PathParams)
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
		assert.Equal(t, "/slug-value/test?param1=abc&param2=123", payload.RawURL)
		assert.Equal(t, http.StatusAccepted, payload.StatusCode)
		assert.Greater(t, payload.Duration, 1000*time.Nanosecond)
		assert.Equal(t, apt.GoGinSDKType, payload.SdkType)

		reqData, _ := json.Marshal(apt.ExampleData2)
		respData, _ := json.Marshal(apt.ExampleDataRedacted)
		assert.Equal(t, reqData, payload.RequestBody)
		assert.Equal(t, respData, payload.ResponseBody)

		publishCalled = true
		return nil
	}

	router := gin.New()
	router.Use(GinMiddleware(client))
	router.POST("/:slug/test", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		assert.NoError(t, err)
		assert.NotEmpty(t, body)
		reqData, _ := json.Marshal(apt.ExampleData2)
		assert.Equal(t, reqData, body)
		c.Header("Content-Type", "application/json")
		c.Header("X-API-KEY", "applicationKey")
		c.JSON(http.StatusAccepted, apt.ExampleData)
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	respData, _ := json.Marshal(apt.ExampleData)
	resp, err := req.Post(ts.URL+"/slug-value/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(apt.ExampleData2),
	)
	assert.NoError(t, err)
	assert.True(t, publishCalled)
	assert.Equal(t, respData, resp.Bytes())
}

func TestGinMiddlewareGET(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := &apt.Client{}
	client.SetConfig(&apt.Config{})

	var publishCalled bool
	respData, _ := json.Marshal(apt.ExampleData)
	client.PublishMessage = func(ctx context.Context, payload apt.Payload) error {
		assert.Equal(t, "GET", payload.Method)
		assert.Equal(t, "/:slug/test", payload.URLPath)
		assert.Equal(t, map[string]string{
			"slug": "slug-value",
		}, payload.PathParams)
		assert.Equal(t, map[string][]string{
			"param1": {"abc"},
			"param2": {"123"},
		}, payload.QueryParams)
		assert.Equal(t, map[string][]string{
			"Accept-Encoding": {"gzip"},
			"User-Agent":      {"Go-http-client/1.1"},
			"X-Api-Key":       {"past-3"},
		}, payload.RequestHeaders)
		assert.Equal(t, map[string][]string{
			"Content-Type": {"application/json"},
		}, payload.ResponseHeaders)
		assert.Equal(t, "/slug-value/test?param1=abc&param2=123", payload.RawURL)
		assert.Equal(t, http.StatusAccepted, payload.StatusCode)
		assert.Greater(t, payload.Duration, 1000*time.Nanosecond)
		assert.Equal(t, []byte{0x6e, 0x75, 0x6c, 0x6c}, payload.RequestBody)
		assert.Equal(t, respData, payload.ResponseBody)
		assert.Equal(t, apt.GoGinSDKType, payload.SdkType)
		publishCalled = true
		return nil
	}
	router := gin.New()
	router.Use(GinMiddleware(client))

	router.GET("/:slug/test", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte{}, body)

		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusAccepted, apt.ExampleData)
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := req.Get(ts.URL+"/slug-value/test",
		req.QueryParam{"param1": "abc", "param2": 123},
		req.Header{
			"X-API-KEY": "past-3",
		},
	)
	assert.NoError(t, err)
	assert.True(t, publishCalled)
	assert.Equal(t, respData, resp.Bytes())
}

func TestOutgoingRequestGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := &apt.Client{}
	client.SetConfig(&apt.Config{})

	var publishCalled bool
	router := gin.New()
	router.Use(GinMiddleware(client))
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
	router.GET("/:slug/test", func(c *gin.Context) {
		hClient := apt.HTTPClient(c.Request.Context(),
			WithRedactHeaders("X-API-KEY"),
			WithRedactRequestBody("$.password"),
			WithRedactResponseBody("$.account_data.account_id"),
		)
		_, _ = hClient.Get("http://localhost:3000/from-gorilla")

		c.JSON(http.StatusAccepted, gin.H{"hello": "world"})
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	_, err := req.Get(ts.URL + "/slug-value/test")
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

	ts := httptest.NewServer(client.Middleware(http.HandlerFunc(handlerFn)))
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

func TestGinMiddlewareGETError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := &apt.Client{}
	client.SetConfig(&apt.Config{})

	var publishCalled bool
	respData, _ := json.Marshal(apt.ExampleData)
	client.PublishMessage = func(ctx context.Context, payload apt.Payload) error {
		publishCalled = true
		return nil
	}
	router := gin.New()
	router.Use(GinMiddleware(client))

	router.GET("/:slug/test", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte{}, body)

		ReportError(c.Request.Context(), errors.New("Test Error"))

		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusAccepted, apt.ExampleData)
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	resp, err := req.Get(ts.URL+"/slug-value/test",
		req.QueryParam{"param1": "abc", "param2": 123},
		req.Header{
			"X-API-KEY": "past-3",
		},
	)
	assert.NoError(t, err)
	assert.True(t, publishCalled)
	assert.Equal(t, respData, resp.Bytes())
}
