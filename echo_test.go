package apitoolkit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/imroc/req"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestEchoServerMiddleware(t *testing.T) {
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
		assert.Equal(t, GoDefaultSDKType, payload.SdkType)

		reqData, _ := json.Marshal(exampleData2)
		respData, _ := json.Marshal(exampleDataRedacted)
		assert.Equal(t, reqData, payload.RequestBody)
		assert.Equal(t, respData, payload.ResponseBody)

		publishCalled = true
		return nil
	}

	e := echo.New()
	e.Use(client.EchoMiddleware)
	e.POST("/:slug/test", func(c echo.Context) (err error) {
		body, err := io.ReadAll(c.Request().Body)
		assert.NoError(t, err)
		assert.NotEmpty(t, body)
		reqData, _ := json.Marshal(exampleData2)
		assert.Equal(t, reqData, body)
		c.Response().Header().Set("Content-Type", "application/json")
		c.Response().Header().Set("X-API-KEY", "applicationKey")
		c.JSON(http.StatusAccepted, exampleData)
		return
	})

	ts := httptest.NewServer(e)
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
	// 0xa is a newline which echo server attaches to the json objects it creates
	respData = append(respData, 0xa)
	assert.Equal(t, respData, resp.Bytes())
}
func TestOutgoingRequestEcho(t *testing.T) {
	client := &Client{
		config: &Config{},
	}
	publishCalled := false
	var parentId *string
	client.PublishMessage = func(ctx context.Context, payload Payload) error {
		if payload.RawURL == "/from-gorilla" {
			assert.NotNil(t, payload.ParentID)
			parentId = payload.ParentID
		} else if payload.URLPath == "/:slug/test" {
			assert.Equal(t, *parentId, payload.MsgID)
		}
		publishCalled = true
		return nil
	}
	router := echo.New()
	router.Use(client.EchoMiddleware)
	router.POST("/:slug/test", func(c echo.Context) (err error) {
		body, err := io.ReadAll(c.Request().Body)
		assert.NotEmpty(t, body)
		reqData, _ := json.Marshal(exampleData2)
		assert.Equal(t, reqData, body)
		HTTPClient := http.DefaultClient
		HTTPClient.Transport = client.WrapRoundTripper(
			c.Request().Context(), HTTPClient.Transport,
		)
		_, _ = HTTPClient.Get("http://localhost:3000/from-gorilla")

		c.JSON(http.StatusAccepted, exampleData)
		return
	})
	ts := httptest.NewServer(router)
	defer ts.Close()

	_, err := req.Post(ts.URL+"/slug-value/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(exampleData2),
	)
	assert.NoError(t, err)
	assert.True(t, publishCalled)

}
