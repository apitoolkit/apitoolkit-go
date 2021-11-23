package main

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestInitializeClient(t *testing.T) {
	client, err := initializeClient(context.Background())
	if err != nil {
		t.Error(err)
	}

	clientType, _ := fmt.Println(reflect.TypeOf(client))
	typeValue, _ := fmt.Println("*pubsub.Client")

	if clientType != typeValue {
		t.Errorf("expected %v but got %v", typeValue, clientType)
	}
}

func TestInitializeTopic(t *testing.T) {
	topic, err := initializeTopic(context.Background())
	if err != nil {
		t.Error(err)
	}

	topicType, _ := fmt.Println(reflect.TypeOf(topic))
	typeValue, _ := fmt.Println("*pubsub.Topic")

	if topicType != typeValue {
		t.Errorf("expected %v but got %v", typeValue, topicType)
	}

	client, err := initializeClient(context.Background())
	if err != nil {
		t.Error(err)
	}
	defer client.Close()

	topicRef := client.Topic(TopicID)

	exists, err := topicRef.Exists(context.Background())
	if err != nil {
		fmt.Println(topicRef.ID())
		t.Error(err)
	}
	
	if !exists {
		t.Error("returned topic instance does not exist when it does")
	}

	topic, err = client.CreateTopic(context.Background(), TopicID)
		if err != nil {
			
		} else {
			fmt.Println(topic.ID())
			t.Error("expected an error but got none")
		}

	
}

func TestPublishMessage(t *testing.T) {
	dat := Data{
		ID:          "testId",
		StatusCode:  "404",
		Body:        "test code",
		RespMessage: "success",
	}

	err := PublishMessage(context.Background(), dat)

	if err != nil {
		t.Error(err)
	}
}
