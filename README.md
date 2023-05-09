[![GoDoc](https://godoc.org/github.com/apitoolkit/apitoolkit-go?status.svg)](https://godoc.org/github.com/apitoolkit/apitoolkit-go)

# API Toolkit Golang Client

The API Toolkit golang client is an sdk used to integrate golang web services with APIToolkit.
It monitors incoming traffic, gathers the requests and sends the request to the apitoolkit servers.

## Design decisions:

- Use the gcp SDK to send real time traffic from REST APIs to the gcp topic

## How to Integrate:

First install the apitoolkit Go sdk:
`go get github.com/apitoolkit/apitoolkit-go`

Then add apitoolkit to your app like so (Gin example):

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

## Client Redacting fields

While it's possible to mark a field as redacted from the apitoolkit dashboard, this client also supports redacting at the client side.
Client side redacting means that those fields would never leave your servers at all. So you feel safer that your sensitive data only stays on your servers.

To mark fields that should be redacted, simply add them to the apitoolkit config struct.
Eg:

```go
func main() {
    apitoolkitCfg := apitoolkit.Config{
        RedactHeaders: []string{"Content-Type", "Authorization", "Cookies"}
        RedactRequestBody: []string{"$.credit-card.cvv", "$.credit-card.name"}
        RedactResponseBody: []string{"$.message.error"}
        APIKey: "<APIKEY>"
    }

  	// Initialize the client using your apitoolkit.io generated apikey
  	apitoolkitClient, _ := apitoolkit.NewClient(context.Background(), apitoolkitCfg)

  	router := gin.New()
  	// Register with the corresponding middleware of your choice. For Gin router, we use the GinMiddleware method.
  	router.Use(apitoolkitClient.GinMiddleware)
  	// Register your handlers as usual and run the gin server as usual.
  	router.POST("/:slug/test", func(c *gin.Context) {c.Text(200, "ok")})
 	...
}
```

It is important to note that while the `RedactHeaders` config field accepts a list of headers(case insensitive),
the RedactRequestBody and RedactResponseBody expect a list of JSONPath strings as arguments.

The choice of JSONPath was selected to allow you have great flexibility in descibing which fields within your responses are sensitive.
Also note that these list of items to be redacted will be aplied to all endpoint requests and responses on your server.
To learn more about jsonpath to help form your queries, please take a look at this cheatsheet:
[https://lzone.de/cheat-sheet/JSONPath](https://lzone.de/cheat-sheet/JSONPath)
