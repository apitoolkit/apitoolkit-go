package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/joho/godotenv"
)

func main() {

	err := publishMessage("pubsub1", "testing1", data{
		messageID:    "1",
		requestBody:  "api request body",
		responseBody: "api response body",
	})
	if err != nil {
		fmt.Println(err)
		return
	}
}

type data struct {
	messageID		string
	requestBody		string
	responseBody	string
}

func initializeClient(ctx context.Context, projectID string) * {
	_ = godotenv.Load(".env")
	client, _ := pubsub.NewClient(ctx, projectID)

	return client
}

func initializeTopic() {

}

// PublishMessage publishes messages to already created topics 
func publishMessage(projectID, topicID string, msg data) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30 * time.Second)
	defer cancel()

	os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8085")

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer client.Close()

	topic := client.Topic(topicID)
	defer topic.Stop()

	exists, err := topic.Exists(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if exists {
		fmt.Println("topic already exist")
	}
	
	if !exists {
		_, err = client.CreateTopic(ctx, topicID)
		fmt.Printf("%s created", topicID)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	msgg := &pubsub.Message{
		ID:              projectID,
		Data:            []byte(fmt.Sprintf(msg.messageID, msg.requestBody, msg.responseBody)),
		PublishTime:     time.Now(),
	}

	_, err = topic.Publish(ctx, msgg).Get(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("published:", string(msgg.Data), msgg.PublishTime)

	return nil
} 

// func publishMessage(projectID, topicID string, msg data) error {
// 	ctx, cancel := context.WithTimeout(context.Background(), 30 * time.Second)
// 	defer cancel()

// 	os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8085")

// 	client, err := pubsub.NewClient(ctx, projectID)
// 	if err != nil {
// 		fmt.Println(err)
// 		return err
// 	}
// 	defer client.Close()

// 	topic := client.Topic(topicID)
// 	defer topic.Stop()

// 	exists, err := topic.Exists(ctx)
// 	if err != nil {
// 		fmt.Println(err)
// 		return err
// 	}

// 	if exists {
// 		fmt.Println("topic already exist")
// 	}
	
// 	if !exists {
// 		_, err = client.CreateTopic(ctx, topicID)
// 		fmt.Printf("%s created", topicID)
// 		if err != nil {
// 			fmt.Println(err)
// 			return err
// 		}
// 	}

// 	msgg := &pubsub.Message{
// 		ID:              projectID,
// 		Data:            []byte(fmt.Sprintf(msg.messageID, msg.requestBody, msg.responseBody)),
// 		PublishTime:     time.Now(),
// 	}

// 	_, err = topic.Publish(ctx, msgg).Get(ctx)
// 	if err != nil {
// 		fmt.Println(err)
// 		return err
// 	}

// 	fmt.Println("published:", string(msgg.Data), msgg.PublishTime)

// 	return nil
// }