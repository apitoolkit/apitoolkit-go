package apitoolkit

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/imroc/req"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestAPIToolkitWorkflow(t *testing.T) {
	// _ = godotenv.Load(".env")
	// client, err := NewClient(context.Background(), Config{RootURL: "http://localhost:8080", APIKey: "x/ZLLsxMNSozzNcf1aZsSzhP9DiURoeev7rlgOhcq20C9t7D"})
	// client, err := NewClient(context.Background(), Config{APIKey: "waAaLZEdNSkzlYdM0aZsTTYc9DmTSoCeuO3s0O0KoDBV9o/D"}) // prod test
	// if !assert.NoError(t, err) {
	// 	t.Fail()
	// 	return
	// }
	// defer client.Close()
	var err error
	client := &Client{
		config: &Config{},
	}

	t.Run("test golang native server middleware", func(t *testing.T) {
		var publishCalled bool
		client.PublishMessage = func(ctx context.Context, payload Payload) error {
			publishCalled = true
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
