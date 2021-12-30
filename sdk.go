package apitoolkit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

const (
	projectID = "past-3"
	topicID   = "apitoolkit-go-client"
)

// data represents request and response details
type data struct {
	Timestamp         time.Time           `json:"timestamp"`
	ProjectID         string              `json:"project_id"`
	Host              string              `json:"host"`
	Method            string              `json:"method"`
	Referer           string              `json:"referer"`
	URLPath           string              `json:"url_path"`
	ProtoMajor        int                 `json:"proto_major"`
	ProtoMinor        int                 `json:"proto_minor"`
	DurationMicroSecs int64               `json:"duration_micro_secs"`
	Duration          time.Duration       `json:"duration"`
	ResponseHeaders   map[string][]string `json:"response_headers"`
	RequestHeaders    map[string][]string `json:"request_headers"`
	RequestBody       []byte              `json:"request_body"`
	ResponseBody      []byte              `json:"response_body"`
	RequestBodyStr    string              `json:"request_body_str"`
	ResponseBodyStr   string              `json:"response_body_str"`

	StatusCode 		  int 				  `json:"status_code"`
}

type Client struct {
	pubsubClient *pubsub.Client
	goReqsTopic  *pubsub.Topic
	config       *Config
}

type Config struct {
	APIKey    string
	ProjectID string
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	_ = godotenv.Load(".env")
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	topic, err := initializeTopic(ctx, client)
	if err != nil {
		return nil, err
	}

	return &Client{
		pubsubClient: client,
		goReqsTopic:  topic,
		config:       &cfg,
	}, nil
}

func (c *Client) Close() error {
	c.goReqsTopic.Stop()
	return c.pubsubClient.Close()
}

// initializeTopic receives the instantiated client object from initialize client and returns a new topic instance
func initializeTopic(ctx context.Context, client *pubsub.Client) (*pubsub.Topic, error) {
	topicRef := client.Topic(topicID)

	exists, err := topicRef.Exists(ctx)
	if err != nil {
		return nil, err
	}

	if exists {
		return topicRef, err
	}

	return client.CreateTopic(ctx, topicID)
}

// PublishMessage publishes payload to a gcp cloud console
func (c *Client) PublishMessage(ctx context.Context, payload data) error {
	if c.goReqsTopic == nil {
		return errors.New("topic is not initialized")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	msgg := &pubsub.Message{
		Data:        data,
		PublishTime: time.Now(),
	}

	c.goReqsTopic.Publish(ctx, msgg)
	return err
}

// "ident":       r.Host,
// 				"method":      r.Method,
// 				"referer":     r.Referer(),
// 				"request_id":  r.Header.Get("X-Request-Id"),
// 				"status_code": record.status,
// 				"url":         r.URL.Path,
// 				"useragent":   r.UserAgent(),
// 				"version":     fmt.Sprintf("%d.%d", r.ProtoMajor, r.ProtoMinor),

// ToolkitMiddleware collects request, response parameters and publishes the payload
func (c *Client) ToolkitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		reqBuf, _ := ioutil.ReadAll(req.Body)
		req.Body.Close()
		req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBuf))

		rec := httptest.NewRecorder()
		start := time.Now()
		next.ServeHTTP(rec, req)

		recRes := rec.Result()
		// io.Copy(res, recRes.Body)
		for k, v := range recRes.Header {
			for _, vv := range v {
				res.Header().Add(k, vv)
			}
		}
		resBody, _ := ioutil.ReadAll(recRes.Body)
		res.WriteHeader(recRes.StatusCode)
		res.Write(resBody)

		since := time.Since(start)
		payload := data{
			Timestamp:         time.Now(),
			ProjectID:         c.config.ProjectID,
			Host:              req.Host,
			Referer:           req.Referer(),
			Method:            req.Method,
			ProtoMajor:        req.ProtoMajor,
			ProtoMinor:        req.ProtoMinor,
			ResponseHeaders:   recRes.Header,
			RequestHeaders:    req.Header,
			RequestBodyStr:    string(reqBuf),
			ResponseBodyStr:   string(resBody),
			RequestBody:       (reqBuf),
			ResponseBody:      (resBody),
			StatusCode:        recRes.StatusCode,
			Duration:          since,
			DurationMicroSecs: since.Microseconds(),
		}

		c.PublishMessage(req.Context(), payload)
	})
}


func (c *Client) GinToolkitMiddleware() gin.HandlerFunc {
	return func(g *gin.Context) {
		reqBuf, _ := ioutil.ReadAll(g.Request.Body)
		g.Request.Body.Close()
		g.Request.Body = ioutil.NopCloser(bytes.NewBuffer(reqBuf))

		rec := httptest.NewRecorder()
		start := time.Now()

		g.Next()

		recRes := rec.Result()
		
		for k, v := range recRes.Header {
			for _, vv := range v {
				g.Header(k, vv)
			}
		}
		resBody, _ := ioutil.ReadAll(recRes.Body)
		g.Render(recRes.StatusCode, render.Data{
			ContentType: "",
			Data:        []byte(resBody),
		})

		since := time.Since(start)
		payload := data{
			Timestamp:		   time.Now(),
			ProjectID:		   c.config.ProjectID,
			Host:              g.Request.Host,
			Referer:           g.Request.Referer(),
			Method:            g.Request.Method,
			ProtoMajor:        g.Request.ProtoMajor,
			ProtoMinor:        g.Request.ProtoMinor,
			ResponseHeaders:   recRes.Header,
			RequestHeaders:    g.Request.Header,
			RequestBodyStr:    string(reqBuf),
			ResponseBodyStr:   string(resBody),
			RequestBody:       (reqBuf),
			ResponseBody:      (resBody),
			StatusCode:        recRes.StatusCode,
			Duration:          since,
			DurationMicroSecs: since.Microseconds(),
		}

		c.PublishMessage(g.Request.Context(), payload)
	}
}

func (c *Client) EchoToolkitMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(e echo.Context) error {
			reqBuf, _ := ioutil.ReadAll(e.Request().Body)
			e.Request().Body.Close()
			e.Request().Body = ioutil.NopCloser(bytes.NewBuffer(reqBuf))

			rec := httptest.NewRecorder()
			start := time.Now()

			recRes := rec.Result()
			
			for k, v := range recRes.Header {
				for _, vv := range v {
					e.Response().Header().Add(k, vv)
				}
			}
			resBody, _ := ioutil.ReadAll(recRes.Body)
			e.Response().WriteHeader(recRes.StatusCode)
			e.Response().Write(resBody)

			since := time.Since(start)
			payload := data{
				Timestamp:		   time.Now(),
				ProjectID:		   c.config.ProjectID,
				Host:              e.Request().Host,
				Referer:           e.Request().Referer(),
				Method:            e.Request().Method,
				ProtoMajor:        e.Request().ProtoMajor,
				ProtoMinor:        e.Request().ProtoMinor,
				ResponseHeaders:   recRes.Header,
				RequestHeaders:    e.Request().Header,
				RequestBodyStr:    string(reqBuf),
				ResponseBodyStr:   string(resBody),
				RequestBody:       (reqBuf),
				ResponseBody:      (resBody),
				StatusCode:        recRes.StatusCode,
				Duration:          since,
				DurationMicroSecs: since.Microseconds(),
			}

			c.PublishMessage(e.Request().Context(), payload)

			return next(e)
		}
	}
}
