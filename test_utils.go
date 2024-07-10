package apitoolkit

var ExampleData = map[string]interface{}{
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

var ExampleDataRedaction = []string{
	"$.status", "$.data.account_data.account_type",
	"$.data.account_data.possible_account_types",
	"$.data.account_data.possible_account_types2[*]",
	"$.non_existent",
}

var ExampleDataRedacted = map[string]interface{}{
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

var ExampleData2 = map[string]interface{}{
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
