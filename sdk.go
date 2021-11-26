package apitoolkit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/joho/godotenv"
)

const (
	TopicID = "apitoolkit-go-client"
)

// data represents request and response details
type data struct {
	ResponseHeader http.Header
	RequestHeader  http.Header
	RequestBody    io.ReadCloser
	ResponseBody   io.ReadCloser
	StatusCode     int
}

type Client struct {
	pubsubClient *pubsub.Client
	goReqsTopic  *pubsub.Topic
	config       *Config
}

type Config struct {
	ProjectID string
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	_ = godotenv.Load(".env")
	client, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, err
	}

	topic, err := initializeTopic(ctx, client)
	if err != nil {
		return nil, err
	}

	return &Client{
		pubsubClient: client,
		goReqsTopic:  topic,
		config:       &cfg,
	}, nil
}

func (c *Client) Close() error {
  c.goReqsTopic.Stop()
	return c.pubsubClient.Close()
}

// initializeTopic receives the instantiated client object from initialize client and returns a new topic instance
func initializeTopic(ctx context.Context, client *pubsub.Client) (*pubsub.Topic, error) {
	topicRef := client.Topic(TopicID)

	exists, err := topicRef.Exists(ctx)
	if err != nil {
		return nil, err
	}

	if exists {
		return topicRef, err
	}

	return client.CreateTopic(ctx, TopicID)
}

// PublishMessage publishes payload to a gcp cloud console
func (c *Client) PublishMessage(ctx context.Context, payload data) error {
	if c.goReqsTopic == nil {
		return errors.New("topic is not initialized")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msgg := &pubsub.Message{
		ID:          c.config.ProjectID,
		Data:        data,
		PublishTime: time.Now(),
	}

	c.goReqsTopic.Publish(ctx, msgg)
	return err
}

// ToolkitMiddleware collects request, response parameters and publishes the payload
func (c *Client) ToolkitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, req)

		io.Copy(res, rec.Result().Body)

		responseHeader := res.Header()
		reqHeader := req.Header

		buf, _ := ioutil.ReadAll(req.Body)
		requestBody := ioutil.NopCloser(bytes.NewBuffer(buf))

		payload := data{
			ResponseHeader: responseHeader,
			RequestHeader:  reqHeader,
			RequestBody:    requestBody,
			ResponseBody:   rec.Result().Body,
			StatusCode:     rec.Result().StatusCode,
		}

		c.PublishMessage(req.Context(), payload)
	})
}
