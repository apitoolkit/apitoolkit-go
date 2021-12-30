package apitoolkit

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/gorilla/mux"
	"github.com/imroc/req"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// func TestInitializeClient(t *testing.T) {
// 	client, err := initializeClient(context.Background())
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	clientType, _ := fmt.Println(reflect.TypeOf(client))
// 	typeValue, _ := fmt.Println("*pubsub.Client")

// 	if clientType != typeValue {
// 		t.Errorf("expected %v but got %v", typeValue, clientType)
// 	}
// }

// func TestInitializeTopic(t *testing.T) {
// 	topic, err := initializeTopic(context.Background())
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	topicType, _ := fmt.Println(reflect.TypeOf(topic))
// 	typeValue, _ := fmt.Println("*pubsub.Topic")

// 	if topicType != typeValue {
// 		t.Errorf("expected %v but got %v", typeValue, topicType)
// 	}

// 	client, err := initializeClient(context.Background())
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer client.Close()

// 	topicRef := client.Topic(TopicID)

// 	exists, err := topicRef.Exists(context.Background())
// 	if err != nil {
// 		fmt.Println(topicRef.ID())
// 		t.Error(err)
// 	}

// 	if !exists {
// 		t.Error("returned topic instance does not exist when it does")
// 	}

// 	topic, err = client.CreateTopic(context.Background(), TopicID)
// 	if err != nil {

// 	} else {
// 		fmt.Println(topic.ID())
// 		t.Error("expected an error but got none")
// 	}
// }

// func TestPublishMessage(t *testing.T) {
// 	msg := data{
// 		StatusCode: 2,
// 	}

// 	err := PublishMessage(context.Background(), msg)

// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// type httpHandler struct{}

// func (hH *httpHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {}

// func TestMiddlewareType(t *testing.T) {
// 	var myH httpHandler
// 	h := ToolkitMiddleware(&myH)

// 	switch v := h.(type) {
// 	case http.Handler:

// 	default:
// 		t.Error(fmt.Sprintf("type is not http.Handler, but is %T", v))
// 	}
// }

// func TestMiddleware(t *testing.T) {
// 	mux := http.NewServeMux()

// 	mux.HandleFunc("/get", func(res http.ResponseWriter, req *http.Request) {

// 		res.Write([]byte("today is a good day"))
// 	})

// 	req := httptest.NewRequest(http.MethodGet, "/get", nil)
// 	res := httptest.NewRecorder()

// 	handler := http.HandlerFunc(func(resp http.ResponseWriter, reqs *http.Request) {})
// 	middleware := ToolkitMiddleware(handler)
// 	middleware.ServeHTTP(res, req)
// }

func TestAPIToolkitWorkflow(t *testing.T) {
	_ = godotenv.Load(".env")
	client, err := NewClient(context.Background(), Config{APIKey: "past-3"})
	assert.NoError(t, err)
	defer client.Close()

	handlerFn := func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		_ = body
		// fmt.Println("HANDLER BODY", string(body))

		jsonByte, err := json.Marshal(exampleData)
		assert.NoError(t, err)

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("X-API-KEY", "applicationKey")
		w.WriteHeader(http.StatusAccepted)

		w.Write(jsonByte)
	}

	ts := httptest.NewServer(client.ToolkitMiddleware(http.HandlerFunc(handlerFn)))
	defer ts.Close()

	r, err := req.Post(ts.URL,
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(exampleData2),
	)
	assert.NoError(t, err)

	fmt.Println(r.Dump())
}

var exampleData = map[string]interface{}{
	"status": "success",
	"data": map[string]interface{}{
		"message": "hello world",
		"account_data": map[string]interface{}{
			"batch_number":           12345,
			"account_id":             "123456789",
			"account_name":           "test account",
			"account_type":           "test",
			"account_status":         "active",
			"account_balance":        "100.00",
			"account_currency":       "USD",
			"account_created_at":     "2020-01-01T00:00:00Z",
			"account_updated_at":     "2020-01-01T00:00:00Z",
			"account_deleted_at":     "2020-01-01T00:00:00Z",
			"possible_account_types": []string{"test", "staging", "production"},
		},
	},
}
var exampleData2 = map[string]interface{}{
	"status": "request",
	"send": map[string]interface{}{
		"message": "hello world",
		"account_data": map[string]interface{}{
			"batch_number":           12345,
			"account_id":             "123456789",
			"account_name":           "test account",
			"account_type":           "test",
			"account_status":         "active",
			"account_balance":        "100.00",
			"account_currency":       "USD",
			"account_created_at":     "2020-01-01T00:00:00Z",
			"account_updated_at":     "2020-01-01T00:00:00Z",
			"account_deleted_at":     "2020-01-01T00:00:00Z",
			"possible_account_types": []string{"test", "staging", "production"},
		},
	},
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAPIGinToolkitWorkflow(t *testing.T) {
	_ = godotenv.Load(".env")
	client, err := NewClient(context.Background(), Config{APIKey: "past-3"})
	assert.NoError(t, err)
	defer client.Close()

	router := gin.New()
	router.Use(client.GinToolkitMiddleware())
	router.POST("/test", func(g *gin.Context) {
		body, _ := ioutil.ReadAll(g.Request.Body)
		_ = body
		// fmt.Println("HANDLER BODY", string(body))

		jsonByte, err := json.Marshal(exampleData)
		assert.NoError(t, err)
		g.Request.Header.Add("Content-Type", "application/json")
		g.Header("X-API-KEY", "applicationKey")
		g.Render(http.StatusAccepted, render.Data{
			ContentType: "",
			Data:        []byte(jsonByte),
		})
	})

	// body := bytes.NewBuffer([]byte("dummy data"))

	ts := httptest.NewServer(router)
	defer ts.Close()

	r, err := req.Post(ts.URL + "/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(exampleData2),
	)
	assert.NoError(t, err)

	fmt.Println(r.Dump())
}

func TestAPIEchoToolkitWorkflow(t *testing.T) {
	_ = godotenv.Load(".env")
	client, err := NewClient(context.Background(), Config{APIKey: "past-3"})
	assert.NoError(t, err)
	defer client.Close()

	router := echo.New()
	router.Use(client.EchoToolkitMiddleware())
	router.POST("/test", func(e echo.Context) error {
		body, _ := ioutil.ReadAll(e.Request().Body)
		_ = body
		// fmt.Println("HANDLER BODY", string(body))

		jsonByte, err := json.Marshal(exampleData)
		assert.NoError(t, err)

		e.Request().Header.Add("Content-Type", "application/json")
		e.Response().WriteHeader(http.StatusAccepted)
		e.Response().Write(jsonByte)
		e.Request().Header.Add("X-API-KEY", "applicationKey")
		
		return err
	})

	// body := bytes.NewBuffer([]byte("dummy data"))

	ts := httptest.NewServer(router)
	defer ts.Close()

	r, err := req.Post(ts.URL + "/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(exampleData2),
	)
	assert.NoError(t, err)

	fmt.Println(r.Dump())
}

func TestAPIGorillaToolkitWorkflow(t *testing.T) {
	_ = godotenv.Load(".env")
	client, err := NewClient(context.Background(), Config{APIKey: "past-3"})
	assert.NoError(t, err)
	defer client.Close()

	router := mux.NewRouter()
	router.Use(client.ToolkitMiddleware)
	router.HandleFunc("/test", func(res http.ResponseWriter, req *http.Request) {
		body, _ := ioutil.ReadAll(req.Body)
		_ = body
		// fmt.Println("HANDLER BODY", string(body))

		jsonByte, err := json.Marshal(exampleData)
		assert.NoError(t, err)

		req.Header.Add("Content-Type", "application/json")
		res.WriteHeader(http.StatusAccepted)
		res.Write(jsonByte)
		req.Header.Add("X-API-KEY", "applicationKey")
	}).Methods("POST")

	// body := bytes.NewBuffer([]byte("dummy data"))

	ts := httptest.NewServer(router)
	defer ts.Close()

	r, err := req.Post(ts.URL + "/test",
		req.Param{"param1": "abc", "param2": 123},
		req.Header{
			"Content-Type": "application/json",
			"X-API-KEY":    "past-3",
		},
		req.BodyJSON(exampleData2),
	)
	assert.NoError(t, err)

	fmt.Println(r.Dump())
}