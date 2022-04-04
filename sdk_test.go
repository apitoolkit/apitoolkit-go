package apitoolkit

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"github.com/joho/godotenv"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestAPIToolkitWorkflow(t *testing.T) {
	_ = godotenv.Load(".env")
	client, err := NewClient(context.Background(), Config{RootURL: "http://localhost:8080", APIKey: "xvUfL8MfaHwzm9YZgqZsGW1L9DnBR9eetbu51L9ZpzxUp4iV"})
	// client, err := NewClient(context.Background(), Config{APIKey: "laVIfc0ZPywzyNMfhaZsS2xJ9GHBTdqeubvtgepdpzkCpt/C"}) // prod test
	if !assert.NoError(t, err) {
		t.Fail()
		return
	}
	defer client.Close()

	t.Run("test golang native server middleware", func(t *testing.T) {
		var publishCalled bool
		client.PublishMessage = func(ctx context.Context, payload Payload) error {
			publishCalled = true
			pretty.Println("payload", payload)
			return nil
		}

		handlerFn := func(w http.ResponseWriter, r *http.Request) {
			body, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.NotEmpty(t, body)

			jsonByte, err := json.Marshal(exampleData)
			assert.NoError(t, err)

			w.Header().Add("Content-Type", "application/json")
			w.Header().Add("X-API-KEY", "applicationKey")
			w.WriteHeader(http.StatusAccepted)
			w.Write(jsonByte)
		}

		ts := httptest.NewServer(client.Middleware(http.HandlerFunc(handlerFn)))
		defer ts.Close()

		_, err = req.Post(ts.URL+"/test",
			req.Param{"param1": "abc", "param2": 123},
			req.Header{
				"Content-Type": "application/json",
				"X-API-KEY":    "past-3",
			},
			req.BodyJSON(exampleData2),
		)
		assert.NoError(t, err)
		assert.True(t, publishCalled)
	})
	t.Run("test gin server middleware", func(t *testing.T) {
		var publishCalled bool
		client.PublishMessage = func(ctx context.Context, payload Payload) error {
			publishCalled = true
			pretty.Println("payload", payload)
			return nil
		}

		router := gin.New()
		router.Use(client.GinMiddleware)
		router.POST("/:slug/test", func(c *gin.Context) {
			body, err := ioutil.ReadAll(c.Request.Body)
			assert.NoError(t, err)
			assert.NotEmpty(t, body)

			c.Header("Content-Type", "application/json")
			c.Header("X-API-KEY", "applicationKey")
			c.JSON(http.StatusAccepted, exampleData)

		})

		router.NoRoute(func(c *gin.Context) {
			fmt.Println("NO ROUTE HANDLER")
		})

		ts := httptest.NewServer(router)
		defer ts.Close()

		fmt.Println("ROOT URL", ts.URL)
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
		pretty.Println(resp.Dump())
	})
	t.Run("test gin server middleware with GET request", func(t *testing.T) {
		var publishCalled bool
		client.PublishMessage = func(ctx context.Context, payload Payload) error {
			publishCalled = true
			pretty.Println("payload", payload)
			return nil
		}

		router := gin.New()
		router.Use(client.GinMiddleware)
		router.GET("/:slug/test", func(c *gin.Context) {
			body, err := ioutil.ReadAll(c.Request.Body)
			assert.NoError(t, err)
			fmt.Println(string(body))

			c.Header("Content-Type", "application/json")
			c.Header("X-API-KEY", "applicationKey")
			c.JSON(http.StatusAccepted, exampleData)
		})

		router.NoRoute(func(c *gin.Context) {
			fmt.Println("NO ROUTE HANDLER")
		})

		ts := httptest.NewServer(router)
		defer ts.Close()

		resp, err := req.Get(ts.URL+"/slug-value/test",
			req.QueryParam{"param1": "abc", "param2": 123},
			req.Header{
				"Content-Type": "application/json",
				"X-API-KEY":    "past-3",
			},
		)
		assert.NoError(t, err)
		assert.True(t, publishCalled)
		pretty.Println(resp.Dump())
	})
}

var exampleData = map[string]interface{}{
	"status": "success",
	"data": map[string]interface{}{
		"message": "hello world",
		"account_data": map[string]interface{}{
			"batch_number":           12345,
			"account_id":             "123456789",
			"account_name":           "test account",
			"account_type":           "test",
			"account_status":         "active",
			"account_balance":        "100.00",
			"account_currency":       "USD",
			"account_created_at":     "2020-01-01T00:00:00Z",
			"account_updated_at":     "2020-01-01T00:00:00Z",
			"account_deleted_at":     "2020-01-01T00:00:00Z",
			"possible_account_types": []string{"test", "staging", "production"},
		},
	},
}
var exampleData2 = map[string]interface{}{
	"status": "request",
	"send": map[string]interface{}{
		"message": "hello world",
		"account_data": map[string]interface{}{
			"batch_number":           12345,
			"account_id":             "123456789",
			"account_name":           "test account",
			"account_type":           "test",
			"account_status":         "active",
			"account_balance":        "100.00",
			"account_currency":       "USD",
			"account_created_at":     "2020-01-01T00:00:00Z",
			"account_updated_at":     "2020-01-01T00:00:00Z",
			"account_deleted_at":     "2020-01-01T00:00:00Z",
			"possible_account_types": []string{"test", "staging", "production"},
		},
	},
}
