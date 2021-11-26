package apitoolkit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
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
	mux := http.NewServeMux()

	mux.HandleFunc("/toolkit-test", func(res http.ResponseWriter, req *http.Request) {
		
		io.WriteString(res, "<html><body><Hello World!</body></html>")
	})

	reader := strings.NewReader("data=dummy request body")
	req := httptest.NewRequest(http.MethodPost, "/toolkit-test", reader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	mux.ServeHTTP(res, req)

	reqHeader := req.Header
	resHeader := res.Header()
	resp := res.Result()
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

	fmt.Println(payload)

	err := PublishMessage(context.Background(), payload)
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

	reader := strings.NewReader("number=2")
	req := httptest.NewRequest(http.MethodPost, "/get", reader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()

	handler := http.HandlerFunc(func(resp http.ResponseWriter, reqs *http.Request) {
		io.WriteString(res, "<html><body><Hello World!</body></html>")
	})
	middleware := ToolkitMiddleware(handler)
	middleware.ServeHTTP(res, req)
}