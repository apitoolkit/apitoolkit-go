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
		utils.Error(http.StatusInternalServerError, w)
		return
	}

	exists, _ := c.TopicExists(data.TopicName, projectID)

	// !exists returns true 
	if !exists {
		utils.Error(http.StatusBadRequest, w)
		return
	}

	_, err = client.CreateTopic(ctx, data.TopicName)
	if err != nil {
		c.Config.Log.Println("Topic could not be created")
		utils.Error(http.StatusInternalServerError, w)
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
		c.Config.Log.Println(err)
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

	topicName := mux.Vars(req)["topic_name"]
	projectID := mux.Vars(req)["project_id"]

	ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Minute)
	defer cancel()

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		c.Config.Log.Fatal(err)
	}
	defer client.Close()

	data := c.Config.Data

	err = utils.ParseJSON(req, &data)
	if err != nil {
		utils.Error(http.StatusInternalServerError, w)
		return
	}

	msg := &pubsub.Message{
		Data: []byte(fmt.Sprintf(data.Message)),
	}

	topic := client.Topic(topicName)

	exists, err := topic.Exists(ctx)
	if err != nil {
		c.Config.Log.Println(err)
	}

	// if exists returns false 
	if exists {
		c.Config.Log.Println("topic does not exist")
		utils.Error(http.StatusBadRequest, w)
		return
	}

	_, err = topic.Publish(ctx, msg).Get(ctx)
	if err != nil {
		utils.Error(http.StatusInternalServerError, w)
		c.Config.Log.Println("could not publish message")
		return
	}

	utils.Success("message published", nil, w)
}