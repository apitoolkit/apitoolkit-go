package apitoolkit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	msg := data {
		StatusCode: 2,
	}

	err := PublishMessage(context.Background(), msg)

	if err != nil {
		t.Error(err)
	}
}

type httpHandler struct{}

func (hH *httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {}

func TestMiddlewareType(t *testing.T) {
	var myH httpHandler
	h := ToolkitMiddleware(&myH)

	switch v := h.(type) {
	case http.Handler:

	default:
		t.Error(fmt.Sprintf("type is not http.Handler, but is %T", v))
	}
}

func TestMiddleware(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/get", func(res http.ResponseWriter, req *http.Request) {
		
		res.Write([]byte("today is a good day"))
	})

	req := httptest.NewRequest(http.MethodGet, "/get", nil)
	res := httptest.NewRecorder()

	handler := http.HandlerFunc(func(resp http.ResponseWriter, reqs *http.Request) {})
	middleware := ToolkitMiddleware(handler)
	middleware.ServeHTTP(res, req)
}