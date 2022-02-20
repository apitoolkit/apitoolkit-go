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
)

// Payload represents request and response details
type Payload struct {
	Timestamp         time.Time           `json:"timestamp"`
	ProjectID         string              `json:"project_id"`
	Host              string              `json:"host"`
	Method            string              `json:"method"`
	Referer           string              `json:"referer"`
	URLPath           string              `json:"url_path"`
	ProtoMajor        int                 `json:"proto_major"`
	ProtoMinor        int                 `json:"proto_minor"`
	DurationMicroSecs int64               `json:"duration_micro_secs"`
	Duration          time.Duration       `json:"duration"`
	ResponseHeaders   map[string][]string `json:"response_headers"`
	RequestHeaders    map[string][]string `json:"request_headers"`
	RequestBody       []byte              `json:"request_body"`
	ResponseBody      []byte              `json:"response_body"`
	StatusCode        int                 `json:"status_code"`
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
			Timestamp:         time.Now(),
			ProjectID:         c.metadata.ProjectId,
			Host:              req.Host,
			Referer:           req.Referer(),
			Method:            req.Method,
			URLPath:           req.URL.Path,
			ProtoMajor:        req.ProtoMajor,
			ProtoMinor:        req.ProtoMinor,
			ResponseHeaders:   recRes.Header,
			RequestHeaders:    req.Header,
			RequestBody:       (reqBuf),
			ResponseBody:      (resBody),
			StatusCode:        recRes.StatusCode,
			Duration:          since,
			DurationMicroSecs: since.Microseconds(),
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

	pretty.Println("params", ctx.Params)
	ctx.Next()

	pretty.Println("params", ctx.Params)

	since := time.Since(start)
	payload := Payload{
		Timestamp:         time.Now(),
		ProjectID:         c.metadata.ProjectId,
		Host:              ctx.Request.Host,
		Referer:           ctx.Request.Referer(),
		Method:            ctx.Request.Method,
		URLPath:           ctx.Request.URL.Path,
		ProtoMajor:        ctx.Request.ProtoMajor,
		ProtoMinor:        ctx.Request.ProtoMinor,
		ResponseHeaders:   ctx.Writer.Header().Clone(),
		RequestHeaders:    ctx.Request.Header,
		RequestBody:       byteBody,
		ResponseBody:      blw.body.Bytes(),
		StatusCode:        ctx.Writer.Status(),
		Duration:          since,
		DurationMicroSecs: since.Microseconds(),
	}

	c.PublishMessage(ctx, payload)
}
