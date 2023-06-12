package apitoolkit

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}


func TestEchoServerMiddleware(t *testing.T) {
		var publishCalled bool
		client.PublishMessage = client.publishMessage

		e := echo.New()
		e.Use(client.EchoMiddleware)

		e.GET("/test/path", func(c echo.Context) (err error) {
			body, err := ioutil.ReadAll(c.Request().Body)
			assert.NoError(t, err)
			fmt.Println(string(body))

			c.Response().Header().Set("Content-Type", "application/json")

			c.Response().Header().Set("X-API-KEY", "applicationKey")
			c.JSON(http.StatusAccepted, exampleData)
			return
		})

		ts := httptest.NewServer(e)
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
}

func TestRedacting(t *testing.T) {
	cfg := Config{
		Debug: true,
	}
	client := Client{
		config: &cfg,
	}

	var publishCalled bool
	client.PublishMessage = func(ctx context.Context, payload Payload) error {
		publishCalled = true
		pretty.Println("🚀  payload", payload)
		pretty.Println("🚀  RequestBody:", string(payload.RequestBody))
		pretty.Println("🚀  ResponseBody:", string(payload.ResponseBody))

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
			"Content-Type":   "application/json",
			"X-API-KEY":      "past-3",
			"X-INPUT-HEADER": "testing",
		},
		req.BodyJSON(exampleData),
	)
	assert.NoError(t, err)
	assert.True(t, publishCalled)
	pretty.Println(resp.Dump())
}

func TestRedactFunc(t *testing.T) {
	t.Run("redact json", func(t *testing.T) {
		exampleJSON, err := json.Marshal(exampleData)
		if err != nil {
			t.Error(err)
		}
		res := redact(exampleJSON, exampleDataRedaction)
		expected, _ := json.Marshal(exampleDataRedacted)
		assert.JSONEq(t, string(expected), string(res))
	})

	t.Run("redactHeaders", func(t *testing.T) {
		result := redactHeaders(map[string][]string{
			"Content-Type": {"application/json"},
			"X-API-KEY":    {"test"},
			"X-rando":      {"test 2"},
		}, []string{"Content-Type", "X-rando"})
		assert.Equal(t, result, map[string][]string{
			"Content-Type": {"[CLIENT_REDACTED]"},
			"X-API-KEY":    {"test"},
			"X-rando":      {"[CLIENT_REDACTED]"},
		})
	})
}

var exampleData = map[string]interface{}{
	"status": "success",
	"data": map[string]interface{}{
		"message": "hello world",
		"account_data": map[string]interface{}{
			"batch_number":            12345,
			"account_id":              "123456789",
			"account_name":            "test account",
			"account_type":            "test",
			"account_status":          "active",
			"account_balance":         "100.00",
			"account_currency":        "USD",
			"account_created_at":      "2020-01-01T00:00:00Z",
			"account_updated_at":      "2020-01-01T00:00:00Z",
			"account_deleted_at":      "2020-01-01T00:00:00Z",
			"possible_account_types":  []string{"test", "staging", "production"},
			"possible_account_types2": []string{"test", "staging", "production"},
		},
	},
}

var exampleDataRedaction = []string{
	"$.status", "$.data.account_data.account_type",
	"$.data.account_data.possible_account_types",
	"$.data.account_data.possible_account_types2[*]",
	"$.non_existent",
}

var exampleDataRedacted = map[string]interface{}{
	"status": "[CLIENT_REDACTED]",
	"data": map[string]interface{}{
		"message": "hello world",
		"account_data": map[string]interface{}{
			"batch_number":            12345,
			"account_id":              "123456789",
			"account_name":            "test account",
			"account_type":            "[CLIENT_REDACTED]",
			"account_status":          "active",
			"account_balance":         "100.00",
			"account_currency":        "USD",
			"account_created_at":      "2020-01-01T00:00:00Z",
			"account_updated_at":      "2020-01-01T00:00:00Z",
			"account_deleted_at":      "2020-01-01T00:00:00Z",
			"possible_account_types":  "[CLIENT_REDACTED]",
			"possible_account_types2": []string{"[CLIENT_REDACTED]", "[CLIENT_REDACTED]", "[CLIENT_REDACTED]"},
		},
	},
}

var exampleData2 = map[string]interface{}{
	"status": "request",
	"send": map[string]interface{}{
		"message": "hello world",
		"account_data": []map[string]interface{}{{
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
		}},
	},
}
