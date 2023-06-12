package apitoolkit

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"github.com/stretchr/testify/assert"
)

func TestGinMiddleware(t *testing.T) {
	client := &Client{
		config: &Config{
			RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
			RedactResponseBody: exampleDataRedaction, 
		},
	}
	var publishCalled bool

	client.PublishMessage = func(ctx context.Context, payload Payload) error {
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
		assert.Equal(t, GoGinSDKType, payload.SdkType)

		reqData, _ := json.Marshal(exampleData2)
		respData, _ := json.Marshal(exampleDataRedacted)
		assert.Equal(t, reqData, payload.RequestBody)
		assert.Equal(t, respData, payload.ResponseBody)

		publishCalled = true
		return nil
	}

	router := gin.New()
	router.Use(client.GinMiddleware)
	router.POST("/:slug/test", func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		assert.NoError(t, err)
		assert.NotEmpty(t, body)
		reqData, _ := json.Marshal(exampleData2)
		assert.Equal(t, reqData, body)
		c.Header("Content-Type", "application/json")
		c.Header("X-API-KEY", "applicationKey")
		c.JSON(http.StatusAccepted, exampleData)
	})

	ts := httptest.NewServer(router)
	defer ts.Close()

	respData, _ := json.Marshal(exampleData)
	resp, err := req.Post(ts.URL+"/slug-value/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(exampleData2),
	)
	assert.NoError(t, err)
	assert.True(t, publishCalled)
	assert.Equal(t, respData, resp.Bytes())
}

func TestGinMiddlewareGET(t *testing.T) {
	client := &Client{
		config: &Config{},
	}
	var publishCalled bool
	respData, _ := json.Marshal(exampleData)
	client.PublishMessage = func(ctx context.Context, payload Payload) error {
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
		assert.Equal(t, GoGinSDKType, payload.SdkType)
		publishCalled = true
		return nil
	}
	router := gin.New()
	router.Use(client.GinMiddleware)

	router.GET("/:slug/test", func(c *gin.Context) {
		body, err := ioutil.ReadAll(c.Request.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte{}, body)

		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusAccepted, exampleData)
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
