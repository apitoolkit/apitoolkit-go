package apitoolkit

import (
	"encoding/json"
	"os"
	"testing"

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
