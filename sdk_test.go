package apitoolkit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/imroc/req"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
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

func TestOutgoingMiddleware(t *testing.T) {
	client := &Client{
		config: &Config{
			RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
			RedactResponseBody: exampleDataRedaction,
		},
	}
	var publishCalled bool
	client.PublishMessage = func(ctx context.Context, payload Payload) error {
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
			assert.Equal(t, GoDefaultSDKType, payload.SdkType)

			reqData, _ := json.Marshal(exampleData2)
			respData, _ := json.Marshal(exampleDataRedacted)

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
			assert.Equal(t, GoOutgoing, payload.SdkType)
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
		jsonByte, err := json.Marshal(exampleData)
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
		req.BodyJSON(exampleData2),
	)
	assert.NoError(t, err)
	assert.True(t, publishCalled)
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
