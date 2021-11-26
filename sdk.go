package apitoolkit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/joho/godotenv"
)

// set TopicID and ProjectID to reflect project use; topic1 and pubsub1 are test values
var (
	TopicID = "topic1"
	ProjectID = "pubsub1"
)

// data represents request and response details
type data struct {
	ResponseHeader		http.Header
	RequestHeader		http.Header
	RequestBody			io.ReadCloser
	ResponseBody		io.ReadCloser
	StatusCode			int
}

// initializeClient creates and return a new pubsub client instance
func initializeClient(ctx context.Context) (*pubsub.Client, error) {
	_ = godotenv.Load(".env")
	client, err := pubsub.NewClient(ctx, ProjectID)
	if err != nil {
		return nil, err
	}

	return client, err
}

// initializeTopic receives the instantiated client object from initialize client and returns a new topic instance
func initializeTopic(ctx context.Context) (*pubsub.Topic, error) {
	client, err := initializeClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	topicRef := client.Topic(TopicID)

	exists, err := topicRef.Exists(ctx)
	if err != nil {
		return nil, err
	}
	
	if exists {
		return topicRef, err
	}

	topic, err := client.CreateTopic(ctx, TopicID)
		if err != nil {
			return nil, err
		}

	return topic, err
}

// initializes the topic instance
var topicInstance, errTopicInstance = initializeTopic(context.Background())

// PublishMessage publishes payload to a gcp cloud console 
func PublishMessage(ctx context.Context, payload data) (error) {

	if errTopicInstance != nil {
		return errTopicInstance
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msgg := &pubsub.Message{
		ID:              ProjectID,
		Data:            data,
		PublishTime:     time.Now(),
	}

	topicInstance.Publish(ctx, msgg)

	return err
}

// ToolkitMiddleware collects request, response parameters and publishes the payload
func ToolkitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, req)
		
		reqHeader := req.Header
		resHeader := res.Header()
		resp := rec.Result()
		body, _ := io.ReadAll(resp.Body)
		responseBody := ioutil.NopCloser(bytes.NewBuffer(body))
		buf, _ := io.ReadAll(req.Body)
		requestBody := ioutil.NopCloser(bytes.NewBuffer(buf))


		payload := data{
			ResponseHeader: resHeader,
			RequestHeader:  reqHeader,
			RequestBody:    requestBody,
			ResponseBody:   responseBody,
			StatusCode:     resp.StatusCode,
		}

		PublishMessage(context.Background(), payload)
	})
}