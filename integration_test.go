//go:build integration

package apitoolkit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/imroc/req"
	"github.com/stretchr/testify/assert"
)

func TestReporting(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		APIKey:             os.Getenv("APITOOLKIT_KEY"),
		RootURL:            "http://localhost:8080",
		RedactHeaders:      []string{"X-Api-Key", "Accept-Encoding"},
		RedactResponseBody: exampleDataRedaction,
		Tags:               []string{"staging"},
	}
	client, err := NewClient(ctx, cfg)
	defer client.Close()
	assert.NoError(t, err)

	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		err1 := errors.Newf("Example Error %v", "value")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"key":"value"}`))
		err2 := errors.Wrap(err1, "wrapper from err2")
		ReportError(r.Context(), err2)
	}

	ts := httptest.NewServer(client.Middleware(http.HandlerFunc(handlerFn)))
	defer ts.Close()

	atHTTPClient := http.DefaultClient
	atHTTPClient.Transport = client.WrapRoundTripper(
		ctx, atHTTPClient.Transport,
		WithRedactHeaders([]string{}),
	)
	req.SetClient(atHTTPClient)
	_, err = req.Post(ts.URL+"/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(exampleData2),
	)
	assert.NoError(t, err)
}
