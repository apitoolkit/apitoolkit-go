package apitoolkit

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"github.com/joho/godotenv"
	"github.com/kr/pretty"
	"google.golang.org/api/option"
)

const (
	topicID = "apitoolkit-go-client"
	SDKType = "go_gin"
)

// Payload represents request and response details
type Payload struct {
	Duration        time.Duration       `json:"duration"`
	Host            string              `json:"host"`
	Method          string              `json:"method"`
	PathParams      map[string]string   `json:"path_params"`
	ProjectID       string              `json:"project_id"`
	ProtoMajor      int                 `json:"proto_major"`
	ProtoMinor      int                 `json:"proto_minor"`
	QueryParams     map[string][]string `json:"query_params"`
	RawURL          string              `json:"raw_url"` // raw request uri: path?query combination
	Referer         string              `json:"referer"`
	RequestBody     []byte              `json:"request_body"`
	RequestHeaders  map[string][]string `json:"request_headers"`
	ResponseBody    []byte              `json:"response_body"`
	ResponseHeaders map[string][]string `json:"response_headers"`
	SdkType         string              `json:"sdk_type"`
	StatusCode      int                 `json:"status_code"`
	Timestamp       time.Time           `json:"timestamp"`
	URLPath         string              `json:"url_path"`
}

type Client struct {
	pubsubClient   *pubsub.Client
	goReqsTopic    *pubsub.Topic
	config         *Config
	metadata       *ClientMetadata
	PublishMessage func(ctx context.Context, payload Payload) error
}

type Config struct {
	RootURL   string
	APIKey    string
	ProjectID string
}

type ClientMetadata struct {
	ProjectId                string          `json:"project_id"`
	PubsubProjectId          string          `json:"pubsub_project_id"`
	PubsubPushServiceAccount json.RawMessage `json:"pubsub_push_service_account"`
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	_ = godotenv.Load(".env")
	url := "https://app.apitoolkit.io"
	if cfg.RootURL != "" {
		url = cfg.RootURL
	}
	resp, err := req.Get(url+"/api/client_metadata", req.Header{
		"Authorization": "Bearer " + cfg.APIKey,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to query apitoolkit for client metadata")
	}

	var clientMetadata ClientMetadata
	err = resp.ToJSON(&clientMetadata)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal client metadata response")
	}

	client, err := pubsub.NewClient(ctx, clientMetadata.PubsubProjectId, option.WithCredentialsJSON(clientMetadata.PubsubPushServiceAccount))
	if err != nil {
		return nil, err
	}

	topic := client.Topic(topicID)
	cl := &Client{
		pubsubClient: client,
		goReqsTopic:  topic,
		config:       &cfg,
		metadata:     &clientMetadata,
	}
	cl.PublishMessage = cl.publishMessage
	return cl, nil
}

func (c *Client) Close() error {
	c.goReqsTopic.Stop()
	return c.pubsubClient.Close()
}

// PublishMessage publishes payload to a gcp cloud console
func (c *Client) publishMessage(ctx context.Context, payload Payload) error {
	if c.goReqsTopic == nil {
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
	return err
}

// Middleware collects request, response parameters and publishes the payload
func (c *Client) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		reqBuf, _ := ioutil.ReadAll(req.Body)
		req.Body.Close()
		req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBuf))

		rec := httptest.NewRecorder()
		start := time.Now()
		next.ServeHTTP(rec, req)

		recRes := rec.Result()
		// io.Copy(res, recRes.Body)
		for k, v := range recRes.Header {
			for _, vv := range v {
				res.Header().Add(k, vv)
			}
		}
		resBody, _ := ioutil.ReadAll(recRes.Body)
		res.WriteHeader(recRes.StatusCode)
		res.Write(resBody)

		since := time.Since(start)
		payload := Payload{
			Duration:        since,
			Host:            req.Host,
			Method:          req.Method,
			PathParams:      nil,
			ProjectID:       c.metadata.ProjectId,
			ProtoMajor:      req.ProtoMajor,
			ProtoMinor:      req.ProtoMinor,
			QueryParams:     req.URL.Query(),
			RawURL:          req.URL.RawPath,
			Referer:         req.Referer(),
			RequestBody:     (reqBuf),
			RequestHeaders:  req.Header,
			ResponseBody:    (resBody),
			ResponseHeaders: recRes.Header,
			SdkType:         SDKType,
			StatusCode:      recRes.StatusCode,
			Timestamp:       time.Now(),
			URLPath:         req.URL.RequestURI(),
		}

		c.PublishMessage(req.Context(), payload)
	})
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	pretty.Println("bodyLogWtiter Write", string(b))
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

func (c *Client) GinMiddleware(ctx *gin.Context) {
	start := time.Now()
	byteBody, _ := ioutil.ReadAll(ctx.Request.Body)
	ctx.Request.Body = ioutil.NopCloser(bytes.NewBuffer(byteBody))

	blw := &bodyLogWriter{body: bytes.NewBuffer([]byte{}), ResponseWriter: ctx.Writer}
	ctx.Writer = blw

	ctx.Next()

	pretty.Println("params", ctx.Params)

	pathParams := map[string]string{}
	for _, param := range ctx.Params {
		pathParams[param.Key] = param.Value
	}

	since := time.Since(start)
	payload := Payload{
		Duration:        since,
		Host:            ctx.Request.Host,
		Method:          ctx.Request.Method,
		ProjectID:       c.metadata.ProjectId,
		ProtoMajor:      ctx.Request.ProtoMajor,
		ProtoMinor:      ctx.Request.ProtoMinor,
		QueryParams:     ctx.Request.URL.Query(),
		PathParams:      pathParams,
		RawURL:          ctx.Request.URL.RequestURI(),
		Referer:         ctx.Request.Referer(),
		RequestBody:     byteBody,
		RequestHeaders:  ctx.Request.Header,
		ResponseBody:    blw.body.Bytes(),
		ResponseHeaders: ctx.Writer.Header().Clone(),
		SdkType:         SDKType,
		StatusCode:      ctx.Writer.Status(),
		Timestamp:       time.Now(),
		URLPath:         ctx.FullPath(),
	}

	c.PublishMessage(ctx, payload)
}
