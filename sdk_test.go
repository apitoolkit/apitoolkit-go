package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/joho/godotenv"
)

func TestInitializeClient(t *testing.T) {
	_ = godotenv.Load(".env")
	_, err := pubsub.NewClient(context.Background(), ProjectID)
	if err != nil {
		t.Error(err)
	}
}

func TestInitializeTopic(t *testing.T) {
	client, _ := pubsub.NewClient(context.Background(), ProjectID)
	defer client.Close()

	topicRef := client.Topic(TopicID)
	defer topicRef.Stop()

	exists, err := topicRef.Exists(context.Background())
	if err != nil {
		t.Error(err)
	}

	if !exists {
		_, err = client.CreateTopic(context.Background(), TopicID)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestObjectJSON(t *testing.T) {
	var result map[string]interface{}

	err := json.Unmarshal([]byte(`{
		"title": "blog post",
		"body": "this is a blog post",
		"likes": "98",
		"comments": "45"
	}`), &result)

	if err != nil {
		t.Error(t)
	}
}

func TestPublishMessage(t *testing.T) {
	var data map[string]interface{}

	json.Unmarshal([]byte(`{
		"title": "blog post",
		"body": "this is a blog post",
		"likes": "98",
		"comments": "45"
	}`), &data)

	client, _ := pubsub.NewClient(context.Background(), ProjectID)
	defer client.Close()

	topicRef := client.Topic(TopicID)

	exists, _ := topicRef.Exists(context.Background())

	msgg := &pubsub.Message{
		ID:              ProjectID,
		Data:            []byte(fmt.Sprintf("payload: %v", data)),
		PublishTime:     time.Now(),
	}

	if !exists {
		topic, _ := client.CreateTopic(context.Background(), TopicID)

		topic.Publish(context.Background(), msgg).Get(context.Background())
		defer topic.Stop()

		fmt.Println(string(msgg.Data))

	} else {
		
		topicRef.Publish(context.Background(), msgg).Get(context.Background())
		defer topicRef.Stop()

		fmt.Println(string(msgg.Data))
	}
}
