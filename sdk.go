// APIToolkit: The API Toolkit golang client is an sdk used to integrate golang web services with APIToolkit.
// It monitors incoming traffic, gathers the requests and sends the request to the apitoolkit servers.
//
// APIToolkit go sdk can be used with most popular Golang routers off the box. And if your routing library of choice is not supported,
// feel free to leave an issue on github, or send in a pul request.
//
// Here's how the SDK can be used with a gin server:
//
//	   // Initialize the client using your apitoolkit.io generated apikey
//	   apitoolkitClient, err := apitoolkit.NewClient(context.Background(), apitoolkit.Config{APIKey: "<APIKEY>"})
//		 if err != nil {
//	    		panic(err)
//		 }
//
//	   router := gin.New()
//
//		 // Register with the corresponding middleware of your choice. For Gin router, we use the GinMiddleware method.
//	   router.Use(apitoolkitClient.GinMiddleware)
//
//	   // Register your handlers as usual and run the gin server as usual.
//	   router.POST("/:slug/test", func(c *gin.Context) {c.Text(200, "ok")})
//	   ...
package apitoolkit

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/AsaiYusuke/jsonpath"
	"github.com/cockroachdb/errors"
	"github.com/imroc/req"
	"github.com/kr/pretty"
	"google.golang.org/api/option"
)

const (
	GoDefaultSDKType = "GoDefault"
	GoGinSDKType     = "GoGin"
)

// Payload represents request and response details
type Payload struct {
	Timestamp       time.Time           `json:"timestamp"`
	RequestHeaders  map[string][]string `json:"request_headers"`
	QueryParams     map[string][]string `json:"query_params"`
	PathParams      map[string]string   `json:"path_params"`
	ResponseHeaders map[string][]string `json:"response_headers"`
	Method          string              `json:"method"`
	SdkType         string              `json:"sdk_type"`
	Host            string              `json:"host"`
	RawURL          string              `json:"raw_url"`
	Referer         string              `json:"referer"`
	ProjectID       string              `json:"project_id"`
	URLPath         string              `json:"url_path"`
	ResponseBody    []byte              `json:"response_body"`
	RequestBody     []byte              `json:"request_body"`
	ProtoMinor      int                 `json:"proto_minor"`
	StatusCode      int                 `json:"status_code"`
	ProtoMajor      int                 `json:"proto_major"`
	Duration        time.Duration       `json:"duration"`
}

type Client struct {
	pubsubClient   *pubsub.Client
	goReqsTopic    *pubsub.Topic
	config         *Config
	metadata       *ClientMetadata
	PublishMessage func(ctx context.Context, payload Payload) error
}

type Config struct {
	Debug bool
	// VerboseDebug should never be enabled in production
	// and logs entire message body which gets sent to APIToolkit
	VerboseDebug bool
	RootURL      string
	APIKey       string
	ProjectID    string
	// A list of field headers whose values should never be sent to apitoolkit
	RedactHeaders      []string
	RedactRequestBody  []string
	RedactResponseBody []string
}

type ClientMetadata struct {
	ProjectId                string          `json:"project_id"`
	PubsubProjectId          string          `json:"pubsub_project_id"`
	TopicID                  string          `json:"topic_id"`
	PubsubPushServiceAccount json.RawMessage `json:"pubsub_push_service_account"`
}

// NewClient would initialize an APIToolkit client which we can use to push data to apitoolkit.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	url := "https://app.apitoolkit.io"
	if cfg.RootURL != "" {
		url = cfg.RootURL
	}

	resp, err := req.Get(url+"/api/client_metadata", req.Header{
		"Authorization": "Bearer " + cfg.APIKey,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to query apitoolkit for client metadata: %w", err)
	}

	var clientMetadata ClientMetadata
	err = resp.ToJSON(&clientMetadata)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal client metadata response: %w", err)
	}

	client, err := pubsub.NewClient(ctx, clientMetadata.PubsubProjectId, option.WithCredentialsJSON(clientMetadata.PubsubPushServiceAccount))
	if err != nil {
		return nil, err
	}

	topic := client.Topic(clientMetadata.TopicID)
	cl := &Client{
		pubsubClient: client,
		goReqsTopic:  topic,
		config:       &cfg,
		metadata:     &clientMetadata,
	}
	cl.PublishMessage = cl.publishMessage

	if cl.config.Debug {
		log.Println("APIToolkit: client initialized successfully")
	}

	return cl, nil
}

// Close cleans up the apitoolkit client. It should be called before the app shorts down, ideally as a defer call.
func (c *Client) Close() error {
	c.goReqsTopic.Stop()
	return c.pubsubClient.Close()
}

// PublishMessage publishes payload to a gcp cloud console
func (c *Client) publishMessage(ctx context.Context, payload Payload) error {
	if c.goReqsTopic == nil {
		if c.config.Debug {
			log.Println("APIToolkit: topic is not initialized. Check client initialization")
		}
		return errors.New("topic is not initialized")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msgg := &pubsub.Message{
		Data:        data,
		PublishTime: time.Now(),
	}

	c.goReqsTopic.Publish(ctx, msgg)
	if c.config.Debug {
		log.Println("APIToolkit: message published to pubsub topic")
		if c.config.VerboseDebug {
			log.Println("APIToolkit: ", pretty.Sprint(data))
		}
	}
	return err
}

// bodyDumpResponseWriter use to preserve the http response body during request processing
type echoBodyLogWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *echoBodyLogWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

func (w *echoBodyLogWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *echoBodyLogWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func (w *echoBodyLogWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

// EchoMiddleware middleware for echo framework, collects requests, response and publishes the payload
func (c *Client) EchoMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) (err error) {
		var reqBuf []byte
		// safely read request body
		if ctx.Request().Body != nil {
			reqBuf, _ = io.ReadAll(ctx.Request().Body)
		}
		ctx.Request().Body = io.NopCloser(bytes.NewBuffer(reqBuf))
		startTime := time.Now()

		// create a MultiWriter that streams the response body into resBody
		resBody := new(bytes.Buffer)
		mw := io.MultiWriter(ctx.Response().Writer, resBody)
		writer := &echoBodyLogWriter{Writer: mw, ResponseWriter: ctx.Response().Writer}
		ctx.Response().Writer = writer

		// pass on request handling
		if err = next(ctx); err != nil {
			ctx.Error(err)
		}

		// proceed post-response processing
		payload := c.buildPayload(GoDefaultSDKType, startTime, ctx.Request(), ctx.Response().Status,
			reqBuf, resBody.Bytes(), ctx.Response().Header(), nil, ctx.Request().URL.RequestURI(),
		)
		c.PublishMessage(ctx.Request().Context(), payload)
		return
	}
}

func (c *Client) buildPayload(SDKType string, trackingStart time.Time, req *http.Request,
	statusCode int, reqBody []byte, respBody []byte, respHeader map[string][]string,
	pathParams map[string]string, urlPath string,
) Payload {
	if req == nil || c == nil || req.URL == nil {
		// Early return with empty payload to prevent any nil pointer panics
		if c.config.Debug {
			log.Println("APIToolkit: nil request or client or url while building payload.")
		}
		return Payload{}
	}
	projectId := ""
	if c.metadata != nil {
		projectId = c.metadata.ProjectId
	}

	redactedHeaders := []string{}
	for _, v := range c.config.RedactHeaders {
		redactedHeaders = append(redactedHeaders, strings.ToLower(v))
	}

	since := time.Since(trackingStart)
	return Payload{
		Duration:        since,
		Host:            req.Host,
		Method:          req.Method,
		PathParams:      pathParams,
		ProjectID:       projectId,
		ProtoMajor:      req.ProtoMajor,
		ProtoMinor:      req.ProtoMinor,
		QueryParams:     req.URL.Query(),
		RawURL:          req.URL.RequestURI(),
		Referer:         req.Referer(),
		RequestBody:     redact(reqBody, c.config.RedactRequestBody),
		RequestHeaders:  redactHeaders(req.Header, redactedHeaders),
		ResponseBody:    redact(respBody, c.config.RedactResponseBody),
		ResponseHeaders: redactHeaders(respHeader, redactedHeaders),
		SdkType:         SDKType,
		StatusCode:      statusCode,
		Timestamp:       time.Now(),
		URLPath:         urlPath,
	}
}

func redact(data []byte, redactList []string) []byte {
	config := jsonpath.Config{}
	config.SetAccessorMode()

	var src interface{}
	json.Unmarshal(data, &src)

	for _, key := range redactList {
		output, _ := jsonpath.Retrieve(key, src, config)
		for _, v := range output {
			accessor, ok := v.(jsonpath.Accessor)
			if ok {
				accessor.Set("[CLIENT_REDACTED]")
			}
		}
	}
	dataJSON, _ := json.Marshal(src)
	return dataJSON
}

func redactHeaders(headers map[string][]string, redactList []string) map[string][]string {
	for k, _ := range headers {
		if find(redactList, k) {
			headers[k] = []string{"[CLIENT_REDACTED]"}
		}
	}
	return headers
}

func find(haystack []string, needle string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}
