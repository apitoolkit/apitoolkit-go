package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/apitoolkit/apitoolkit-go-client/conf"
	"github.com/apitoolkit/apitoolkit-go-client/utils"
	"github.com/gorilla/mux"
)

var Repo *Repository

type Repository struct {
	Config 	*conf.Configuration
}

// CreateTopic creates the topic where messages are to be published 
func(c *Repository) CreateTopic(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Minute)
	defer cancel()

	projectID := mux.Vars(req)["project_id"]

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		c.Config.Log.Fatal(err)
	}
	defer client.Close()

	data := c.Config.Data

	err = utils.ParseJSON(req, &data)
	if err != nil {
		utils.Error(http.StatusBadRequest, w)
		return
	}

	exists, _ := c.TopicExists(data.TopicName, projectID)

	if exists {
		utils.Error(http.StatusBadRequest, w)
		return
	}

	_, err = client.CreateTopic(ctx, data.TopicName)
	if err != nil {
		c.Config.Log.Println("Topic could not be created")
		utils.Error(http.StatusBadRequest, w)
		return
	}

	utils.Success("topic created", nil, w)
}

// TopicExists checks if message topic had already been created 
func(c *Repository) TopicExists(name, projectID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Minute)
	defer cancel()

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		c.Config.Log.Fatal(err)
	}
	defer client.Close()

	topic := client.Topic(name)

	exists, err := topic.Exists(ctx)
	if err != nil {
		c.Config.Log.Fatal(err)
	}

	if exists {
		c.Config.Log.Println("topic already exists")
		return true, nil
	}

	return false, nil
}

// PublishMessage publishes messages to already created topics
func(c *Repository) PublishMessage(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Minute)
	defer cancel()

	// topicName := mux.Vars(req)["topic_name"]

	data := c.Config.Data

	err := utils.ParseJSON(req, &data)
	if err != nil {
		utils.Error(http.StatusBadRequest, w)
		return
	}

	msg := &pubsub.Message{
		Data: []byte(fmt.Sprintf(data.Message)),
	}

	var topic *pubsub.Topic

	_, err = topic.Publish(ctx, msg).Get(ctx)
	if err != nil {
		utils.Error(http.StatusBadRequest, w)
		c.Config.Log.Println("could not publish message")
	}

	utils.Success("message published", nil, w)
}