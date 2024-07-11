package apitoolkit_tlsclient

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	apitoolkit "github.com/apitoolkit/apitoolkit-go"
	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

func TestTlsClient(t *testing.T) {
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_120),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar), // create cookieJar instance and pass it as argument
	}

	clientTLS, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()

	conf := apitoolkit.Config{
		Debug:        true,
		VerboseDebug: true,
		APIKey:       os.Getenv("APITOOLKIT_KEY"),
	}
	atClient, err := apitoolkit.NewClient(ctx, conf)
	if err != nil {
		t.Error(err)
	}

	client := NewHttpClient(ctx, clientTLS, atClient)

	req, err := fhttp.NewRequest(http.MethodGet, "https://jsonplaceholder.typicode.com/posts/1", nil)
	if err != nil {
		t.Error(err)
	}

	req.Header = fhttp.Header{
		"accept":          {"*/*"},
		"accept-language": {"de-DE,de;q=0.9,en-US;q=0.8,en;q=0.7"},
		"user-agent":      {"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"},
		fhttp.HeaderOrderKey: {
			"accept",
			"accept-language",
			"user-agent",
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
	}

	defer resp.Body.Close()

	log.Printf("status code: %d", resp.StatusCode)

	readBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}

	log.Println(string(readBytes))
}
