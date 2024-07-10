package apitoolkitfiber

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apt "github.com/apitoolkit/apitoolkit-go"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestFiberMiddleware(t *testing.T) {
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
			// "Accept-Encoding": {"gzip"},
			"Content-Length": {"437"},
			"Content-Type":   {"application/json"},
			// "User-Agent":      {"Go-http-client/1.1"},
			"X-Api-Key": {"past-3"},
			"Host":      {"example.com"},
		}, payload.RequestHeaders)
		assert.Equal(t, map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}}, payload.ResponseHeaders)
		assert.Equal(t, "/slug-value/test?param1=abc&param2=123", payload.RawURL)
		assert.Equal(t, http.StatusAccepted, payload.StatusCode)
		assert.Greater(t, payload.Duration, 1000*time.Nanosecond)
		assert.Equal(t, apt.GoFiberSDKType, payload.SdkType)

		reqData, _ := json.Marshal(apt.ExampleData2)
		respData, _ := json.Marshal(apt.ExampleDataRedacted)
		assert.Equal(t, reqData, payload.RequestBody)
		assert.Equal(t, respData, payload.ResponseBody)

		publishCalled = true
		return nil
	}

	router := fiber.New()
	router.Use(FiberMiddleware(client))
	router.Post("/:slug/test", func(c *fiber.Ctx) error {
		body := c.Request().Body()
		assert.NotEmpty(t, body)
		reqData, _ := json.Marshal(apt.ExampleData2)
		assert.Equal(t, reqData, body)

		c.Set("Content-Type", "application/json")
		c.Set("X-API-KEY", "applicationKey")

		return c.Status(http.StatusAccepted).JSON(apt.ExampleData)
	})

	// ts := httptest.NewServer(router.Server())
	// defer ts.Close()

	respData, _ := json.Marshal(apt.ExampleData)
	// resp, err := req.Post(ts.URL+"/slug-value/test",
	// 	req.Param{"param1": "abc", "param2": 123},
	// 	req.Header{
	// 		"Content-Type": "application/json",
	// 		"X-API-KEY":    "past-3",
	// 	},
	// 	req.BodyJSON(apt.ExampleData2),
	// )

	reqData, _ := json.Marshal(apt.ExampleData2)
	req := httptest.NewRequest("POST", "/slug-value/test?param1=abc&param2=123", bytes.NewReader(reqData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", "past-3")

	resp, err := router.Test(req, 10)
	assert.NoError(t, err)
	assert.True(t, publishCalled)
	data, err := ioutil.ReadAll(resp.Body)
	assert.Equal(t, respData, data)
}

func TestOutgoingRequestFiber(t *testing.T) {
	client := &apt.Client{}
	client.SetConfig(&apt.Config{})
	publishCalled := false
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
	router := fiber.New()
	router.Use(FiberMiddleware(client))
	router.Post("/:slug/test", func(c *fiber.Ctx) error {
		body := c.Request().Body()
		assert.NotEmpty(t, body)
		reqData, _ := json.Marshal(apt.ExampleData2)
		assert.Equal(t, reqData, body)
		HTTPClient := http.DefaultClient
		HTTPClient.Transport = client.WrapRoundTripper(
			c.UserContext(), HTTPClient.Transport,
		)
		_, _ = HTTPClient.Get("http://localhost:3000/from-gorilla")

		c.Append("Content-Type", "application/json")
		c.Append("X-API-KEY", "applicationKey")

		return c.Status(http.StatusAccepted).JSON(apt.ExampleData)
	})

	reqData, _ := json.Marshal(apt.ExampleData2)
	ts := httptest.NewRequest("POST", "/slug-value/test?param1=abc&param2=123", bytes.NewReader(reqData))
	ts.Header.Set("Content-Type", "application/json")
	ts.Header.Set("X-API-KEY", "past-3")

	_, err := router.Test(ts)
	assert.NoError(t, err)
	assert.True(t, publishCalled)
}
