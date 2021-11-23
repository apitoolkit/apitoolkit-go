package main

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/joho/godotenv"
)

// set TopicID and ProjectID to reflect project use; topic1 and pubsub1 are test values
var (
	TopicID = "topic1"
	ProjectID = "pubsub1"
)

// Data represents test request and response struct
type Data struct {
	ID			string
	StatusCode	string
	Body		string
	RespMessage	string  
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


// PublishMessage publishes payload to a gcp cloud console 
func PublishMessage(ctx context.Context, payload Data) (error) {
	topic, err := initializeTopic(ctx)
	if err != nil {
		return err
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

	topic.Publish(ctx, msgg)

	return err
}