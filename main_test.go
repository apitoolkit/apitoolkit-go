package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
)

func TestMain(t *testing.T) {
	err := publishMessage("pubsub1", "testing1", data{
		messageID:    "1",
		requestBody:  "api request body",
		responseBody: "api response body",
	})

	if err != nil {
		t.Error("publish message function returned an error")
	}
}

func TestPublishMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30 * time.Second)
	defer cancel()

	os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8085")
	testEnv := os.Getenv("PUBSUB_EMULATOR_HOST")
	if testEnv != "localhost:8085" {
		t.Error("env error")
	}

	client, err := pubsub.NewClient(ctx, "pubsub1")
	if err != nil {
		t.Error(err)
	}
	defer client.Close()

	topic := client.Topic("testing1")
	defer topic.Stop()

	exists, err := topic.Exists(ctx)
	if err != nil {
		t.Error(err)
	}

	if !exists {
		t.Error("says topic does not exist when it does exist")
		_, err = client.CreateTopic(ctx, "testing1")
		if err != nil {
			t.Error(err)
		}
	}

	msg := data{
		messageID:    "1",
		requestBody:  "api request body",
		responseBody: "api response body",
		}

	msgg := &pubsub.Message{
		ID:              "pubsub1",
		Data:            []byte(fmt.Sprintf(msg.messageID, msg.requestBody, msg.responseBody)),
		PublishTime:     time.Now(),
	}

	_, err = topic.Publish(ctx, msgg).Get(ctx)
	if err != nil {
		t.Error(err)
	}

	input := []byte(fmt.Sprintf(msg.messageID, msg.requestBody, msg.responseBody))
	output := msgg.Data

	inputS := string(input)
	outputS := string(output)

	if inputS == outputS {
	} else {
		t.Errorf("expected %v but got %v", inputS, outputS)
	}
}