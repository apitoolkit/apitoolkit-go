# API Toolkit Golang Client
The API Toolkit golang client is an sdk used to integrate golang web services with APIToolkit. 
It monitors incoming traffic, gathers the requests and sends the request to the apitoolkit servers.

## Design decisions:
- Use the gcp SDK to send real time traffic from REST APIs to the gcp topic

## How to Integrate:
Gin example:

```go
package main

import (
  	// Import the apitoolkit golang sdk
  	apitoolkit "github.com/apitoolkit/apitoolkit-go"
)

func main() {
  	// Initialize the client using your apitoolkit.io generated apikey
  	apitoolkitClient, err := apitoolkit.NewClient(context.Background(), apitoolkit.Config{APIKey: "<APIKEY>"})
	if err != nil {
    		panic(err)
	}

  	router := gin.New()

  	// Register with the corresponding middleware of your choice. For Gin router, we use the GinMiddleware method.
  	router.Use(apitoolkitClient.GinMiddleware)

  	// Register your handlers as usual and run the gin server as usual.
  	router.POST("/:slug/test", func(c *gin.Context) {c.Text(200, "ok")})
 	...
}

```
