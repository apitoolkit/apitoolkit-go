[![GoDoc](https://godoc.org/github.com/apitoolkit/apitoolkit-go?status.svg)](https://godoc.org/github.com/apitoolkit/apitoolkit-go)

# API Toolkit Golang Client

The API Toolkit golang client is an sdk used to integrate golang web services with APIToolkit.
It monitors incoming traffic, gathers the requests and sends the request to the apitoolkit servers.

## Design decisions:

- Use the gcp SDK to send real time traffic from REST APIs to the gcp topic

## How to Integrate:

First install the apitoolkit Go sdk:
`go get github.com/apitoolkit/apitoolkit-go`

### Gin web server integration
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

### Fiber Router server integration
```go
package main

import (
  	// Import the apitoolkit golang sdk
    apitoolkit "github.com/apitoolkit/apitoolkit-go"
    fiber "github.com/gofiber/fiber/v2"
)

func main() {
  	// Initialize the client using your apitoolkit.io generated apikey
  	apitoolkitClient, err := apitoolkit.NewClient(context.Background(), apitoolkit.Config{APIKey: "<APIKEY>"})
	if err != nil {
    		panic(err)
	}

  	router := fiber.New()

  	// Register with the corresponding middleware of your choice. For Gin router, we use the GinMiddleware method.
  	router.Use(apitoolkitClient.FiberMiddleware)

  	// Register your handlers as usual and run the gin server as usual.
  	router.Post("/:slug/test", func(c *fiber.Ctx) {c.Status(200).JSON({"status":"ok"})})
 	...
}
```

### Echo Router server integration
```go
package main

import (
  	// Import the apitoolkit golang sdk
    apitoolkit "github.com/apitoolkit/apitoolkit-go"
    echo "github.com/labstack/echo/v4"
)

func main() {
  	// Initialize the client using your apitoolkit.io generated apikey
  	apitoolkitClient, err := apitoolkit.NewClient(context.Background(), apitoolkit.Config{APIKey: "<APIKEY>"})
	if err != nil {
    		panic(err)
	}

  	router := echo.New()

  	// Register with the corresponding middleware of your choice. For Gin router, we use the GinMiddleware method.
  	router.Use(apitoolkitClient.EchoMiddleware)

  	// Register your handlers as usual and run the gin server as usual.
  	router.POST("/:slug/test", func(c *fiber.Ctx) {c.JSON(200, {"status":"ok"})})
 	...
}
```


### Gorilla Mux Golang HTTP router integration 
```go
import (
  	// Import the apitoolkit golang sdk
    apitoolkit "github.com/apitoolkit/apitoolkit-go"
    "github.com/gorilla/mux"
)

func main() {
  	// Initialize the client using your apitoolkit.io generated apikey
  	apitoolkitClient, err := apitoolkit.NewClient(context.Background(), apitoolkit.Config{APIKey: "<APIKEY>"})
	if err != nil {
        panic(err)
	}

  	router := mux.NewRouter()
  	// Register with the corresponding middleware of your choice. For Gin router, we use the GinMiddleware method.
  	router.Use(apitoolkitClient.GorillaMuxMiddleware)

  	// Register your handlers as usual and run the gin server as usual.
  	router.HandleFunc("/{slug:[a-z]+}/test", func(w http.ResponseWriter, r *http.Request) {..}).Methods(http.MethodPost)
 	...
}
```


### Golang Chi HTTP router integration 
```go
import (
    apitoolkit "github.com/apitoolkit/apitoolkit-go"
)

func main() {
  	// Initialize the client using your apitoolkit.io generated apikey
  	apitoolkitClient, err := apitoolkit.NewClient(context.Background(), apitoolkit.Config{APIKey: "<APIKEY>"})
	if err != nil {
        panic(err)
	}

  	router := chi.NewRouter()
  	// Register with the corresponding middleware of your choice. For Gin router, we use the GinMiddleware method.
  	router.Use(apitoolkitClient.ChiMiddleware)

  	// Register your handlers as usual and run the gin server as usual.
  	router.Post("/{slug:[a-z]+}/test", func(w http.ResponseWriter, r *http.Request) {..})
 	...
}
```

### Native Golang HTTP router integration 
Only use this as a last resort. Make a request via github issues if your routing library of choice is not supported.
```go
package main

import (
  	// Import the apitoolkit golang sdk
    apitoolkit "github.com/apitoolkit/apitoolkit-go"
    "github.com/gorilla/mux"
)

func main() {
  	// Initialize the client using your apitoolkit.io generated apikey
  	apitoolkitClient, err := apitoolkit.NewClient(context.Background(), apitoolkit.Config{APIKey: "<APIKEY>"})
	if err != nil {
    		panic(err)
	}

  	// Register with the corresponding middleware of your choice. For Gin router, we use the GinMiddleware method.
  	router.Use(apitoolkitClient.Middleware)

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
        	RedactHeaders: []string{"Content-Type", "Authorization", "Cookies"},
        	RedactRequestBody: []string{"$.credit-card.cvv", "$.credit-card.name"},
        	RedactResponseBody: []string{"$.message.error"},
        	APIKey: "<APIKEY>",
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

## Outgoing Requests

Access an instrumented http client which will 
automatically monitor requests made to third parties. These outgoing requests can be monitored from the apitoolkit dashboard

```go
    httpClient := apitoolkit.HTTPClient(ctx)
    httpClient.Post(..) // Use the client like any regular golang http client. 

```

You can redact fields using functional options to specify what fields to redact. 
eg. 

```go
    httpClient := apitoolkit.HTTPClient(ctx, apitoolkit.WithRedactHeaders("ABC", "$abc.xyz"))
    httpClient.Post(..) // Use the client like any regular golang http client. 
```

## Report Errors

If you've used sentry, or bugsnag, or rollbar, then you're already familiar with this usecase.
But you can report an error to apitoolkit. A difference, is that errors are always associated with a parent request, and helps you query and associate the errors which occured while serving a given customer request. To request errors to APIToolkit use call the `ReportError` method of `apitoolkit` not the client returned by `apitoolkit.NewClient` with the request context and the error to report
Examples:

**Native Go**

```go
    file, err := os.Open("non-existing-file.txt")
    if err != nil {
        // Report the error to apitoolkit
        // Ensure that the ctx is the context which is passed down from the handlers.
        apitoolkit.ReportError(ctx, err)
    }

```
