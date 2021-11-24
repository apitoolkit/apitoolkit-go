package apitoolkit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/joho/godotenv"
)

// set TopicID and ProjectID to reflect project use; topic1 and pubsub1 are test values
var (
	TopicID = "topic1"
	ProjectID = "pubsub1"
)

// Data represents request and response details
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

// ToolkitMiddleware collects request and response parameters
func ToolkitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		// responseHeader := req.Response.Header
		reqHeader := req.Header
		reqBody := req.Body
		// resBody := req.Response.Body
		// statusCode := req.Response.StatusCode

		payload := data {
			// ResponseHeader: responseHeader,
			RequestHeader: reqHeader,
			RequestBody: reqBody,
			// ResponseBody: resBody,
			// StatusCode: statusCode,
		}

		PublishMessage(context.Background(), payload)

		fmt.Println(payload)
		next.ServeHTTP(res, req)
	})
}

