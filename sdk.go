// APIToolkit: The API Toolkit golang client is an sdk used to integrate golang web services with APIToolkit.
// It monitors incoming traffic, gathers the requests and sends the request to the apitoolkit servers.
//
// APIToolkit go sdk can be used with most popular Golang routers off the box. And if your routing library of choice is not supported,
// feel free to leave an issue on github, or send in a pull request.
//
// Here's how the SDK can be used with a gin server:
// ```go
//
//	func main(){
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
//	   router.POST("/:slug/test", func(c *gin.Context) {c.String(200, "ok")})
//	}
//
// ```
package apitoolkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/AsaiYusuke/jsonpath"
	"github.com/google/uuid"
	"github.com/imroc/req"
	"github.com/kr/pretty"
	"github.com/valyala/fasthttp"
	"google.golang.org/api/option"
)

const (
	GoDefaultSDKType = "GoBuiltIn"
	GoGinSDKType     = "GoGin"
	GoGorillaMux     = "GoGorillaMux"
	GoOutgoing       = "GoOutgoing"
	GoFiberSDKType   = "GoFiber"
)

// Payload represents request and response details
// FIXME: How would we handle errors from background processes (Not web requests)
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
	Errors          []ATError           `json:"errors"`
	ServiceVersion  *string             `json:"service_version"`
	Tags            []string            `json:"tags"`
	MsgID           string              `json:"msg_id"`
	ParentID        *string             `json:"parent_id"`
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
	// ServiceVersion is an identifier to help you track deployments. This could be a semver version or a git hash or anything you like.
	ServiceVersion string
	// A list of field headers whose values should never be sent to apitoolkit
	RedactHeaders      []string
	RedactRequestBody  []string
	RedactResponseBody []string
	// Tags are arbitrary identifiers for service being tracked, and can be used as filters on apitoolkit.
	Tags []string `json:"tags"`
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
	if resp.Response().StatusCode >= 400 {
		return nil, fmt.Errorf("unable to authenticate APIKey against apitoolkit servers. Is your API Key correct?")
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
	if c.goReqsTopic != nil {
		c.goReqsTopic.Stop()
		return c.pubsubClient.Close()

	}
	return nil
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

func (c *Client) BuildPayload(SDKType string, trackingStart time.Time, req *http.Request,
	statusCode int, reqBody []byte, respBody []byte, respHeader map[string][]string,
	pathParams map[string]string, urlPath string,
	redactHeadersList,
	redactRequestBodyList, redactResponseBodyList []string,
	errorList []ATError,
	msgID uuid.UUID,
	parentID *uuid.UUID,
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

	redactedHeaders := []string{"password", "Authorization", "Cookies"}
	for _, v := range redactHeadersList {
		redactedHeaders = append(redactedHeaders, strings.ToLower(v))
	}

	since := time.Since(trackingStart)
	var parentIDVal *string
	if parentID != nil {
		parentIDStr := (*parentID).String()
		parentIDVal = &parentIDStr
	}

	var serviceVersion *string
	if c.config.ServiceVersion != "" {
		serviceVersion = &c.config.ServiceVersion
	}
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
		RequestBody:     redact(reqBody, redactRequestBodyList),
		RequestHeaders:  redactHeaders(req.Header, redactedHeaders),
		ResponseBody:    redact(respBody, redactResponseBodyList),
		ResponseHeaders: redactHeaders(respHeader, redactedHeaders),
		SdkType:         SDKType,
		StatusCode:      statusCode,
		Timestamp:       time.Now(),
		URLPath:         urlPath,
		Errors:          errorList,
		ServiceVersion:  serviceVersion,
		Tags:            c.config.Tags,
		MsgID:           msgID.String(),
		ParentID:        parentIDVal,
	}
}

func (c *Client) buildFastHTTPPayload(SDKType string, trackingStart time.Time, req *fasthttp.RequestCtx,
	statusCode int, reqBody []byte, respBody []byte, respHeader map[string][]string,
	pathParams map[string]string, urlPath string,
	redactHeadersList,
	redactRequestBodyList, redactResponseBodyList []string,
	errorList []ATError,
	msgID uuid.UUID,
	parentID *uuid.UUID,
	referer string,
) Payload {
	if req == nil || c == nil || req.URI() == nil {
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

	queryParams := map[string][]string{}
	req.QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = []string{string(value)}
	})

	reqHeaders := map[string][]string{}
	req.Request.Header.VisitAll(func(key, value []byte) {
		reqHeaders[string(key)] = []string{string(value)}
	})

	redactedHeaders := []string{"password", "Authorization", "Cookies"}
	for _, v := range redactHeadersList {
		redactedHeaders = append(redactedHeaders, strings.ToLower(v))
	}

	since := time.Since(trackingStart)
	var parentIDVal *string
	if parentID != nil {
		parentIDStr := (*parentID).String()
		parentIDVal = &parentIDStr
	}

	var serviceVersion *string
	if c.config.ServiceVersion != "" {
		serviceVersion = &c.config.ServiceVersion
	}
	return Payload{
		Duration:        since,
		Host:            string(req.Host()),
		Method:          string(req.Method()),
		PathParams:      pathParams,
		ProjectID:       projectId,
		ProtoMajor:      1, // req.ProtoMajor,
		ProtoMinor:      1, // req.ProtoMinor,
		QueryParams:     queryParams,
		RawURL:          string(req.RequestURI()),
		Referer:         referer,
		RequestBody:     redact(reqBody, redactRequestBodyList),
		RequestHeaders:  redactHeaders(reqHeaders, redactedHeaders),
		ResponseBody:    redact(respBody, redactResponseBodyList),
		ResponseHeaders: redactHeaders(respHeader, redactedHeaders),
		SdkType:         SDKType,
		StatusCode:      statusCode,
		Timestamp:       time.Now(),
		URLPath:         urlPath,
		Errors:          errorList,
		ServiceVersion:  serviceVersion,
		Tags:            c.config.Tags,
		MsgID:           msgID.String(),
		ParentID:        parentIDVal,
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
	for k := range headers {
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
