package apitoolkit

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

	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"github.com/stretchr/testify/assert"
)

func TestErrorReporting(t *testing.T) {
	client := &Client{
		config: &Config{
			RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
			RedactResponseBody: exampleDataRedaction,
		},
	}
	var publishCalled bool
	client.PublishMessage = func(ctx context.Context, payload Payload) error {
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

	outClient := &Client{
		config: &Config{},
	}

	outClient.PublishMessage = func(ctx context.Context, payload Payload) error {
		assert.Equal(t, "/test?param1=abc&param2=123", payload.RawURL)
		assert.Equal(t, http.StatusAccepted, payload.StatusCode)
		assert.Greater(t, payload.Duration, 1000*time.Nanosecond)
		assert.Equal(t, GoOutgoing, payload.SdkType)
		return nil
	}

	_, err := req.Post(ts.URL+"/test",
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

func TestGinMiddlewareGETError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := &Client{
		config: &Config{},
	}
	var publishCalled bool
	respData, _ := json.Marshal(exampleData)
	client.PublishMessage = func(ctx context.Context, payload Payload) error {
		publishCalled = true
		return nil
	}
	router := gin.New()
	router.Use(client.GinMiddleware)

	router.GET("/:slug/test", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		assert.NoError(t, err)
		assert.Equal(t, []byte{}, body)

		ReportError(c.Request.Context(), errors.New("Test Error"))

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
