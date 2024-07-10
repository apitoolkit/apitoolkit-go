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
		exampleJSON, err := json.Marshal(ExampleData)
		if err != nil {
			t.Error(err)
		}
		res := RedactJSON(exampleJSON, ExampleDataRedaction)
		expected, _ := json.Marshal(ExampleDataRedacted)
		assert.JSONEq(t, string(expected), string(res))
	})

	t.Run("redactHeaders", func(t *testing.T) {
		result := RedactHeaders(map[string][]string{
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
