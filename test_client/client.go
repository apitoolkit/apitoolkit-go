package client

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"cloud.google.com/go/pubsub"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

// for possible auth
// func getEnv(key string) string {
// 	if value, exists := os.LookupEnv(key); exists {
// 		return value
// 	}

// 	return ""
// }

func NewClient(ctx context.Context) (*pubsub.Client, error) {
	_ = godotenv.Load("path to .env")

	// for possible auth 
	// service_id := getEnv("serviceID")
	// service_pin := getEnv("servicePin")

	// resp, err := http.Get(fmt.Sprintf("http://localhost:8000/testserver/%s/%s", service_id, service_pin))

	resp, err := http.Get("http://localhost:8000/testserver")
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	if err = ioutil.WriteFile("test_cred.json", respData, 0644); err != nil {
		fmt.Println(err)
	}

	// I need real cred files to be able to fully test this part
	client, err := pubsub.NewClient(ctx, "past-3", option.WithCredentialsFile("path to test_cred.json"))
	// client, err := pubsub.NewClient(ctx, "past-3")
	if err != nil {
		return nil, err
	}

	defer client.Close()

	fmt.Println("client initialized")

	return client, err
}
