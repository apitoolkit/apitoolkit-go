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

```go
    ctx := context.Background()
    HTTPClient := http.DefaultClient
    HTTPClient.Transport = apitoolkitClient.WrapRoundTripper(
        ctx, HTTPClient.Transport,
        WithRedactHeaders([]string{}),
    )

```

The code above shows how to use the custom roundtripper to replace the transport in the default http client.
The resulting HTTP client can be used for any purpose, but will send a copy of all incoming and outgoing requests
to the apitoolkit servers. So to allow monitoring outgoing request from your servers use the `HTTPClient` to make http requests.

## Report Errors

If you've used sentry, or bugsnag, or rollbar, then you're already familiar with this usecase.
But you can report an error to apitoolkit. A difference, is that errors are always associated with a parent request, and helps you query and associate the errors which occured while serving a given customer request. To request errors to APIToolkit use call the `ReportError` method of `apitoolkit` not the client returned by `apitoolkit.NewClient` with the request context and the error to report
Examples:

**Native Go**

```go
package main

import (
	"fmt"
	"net/http"
	apitoolkit "github.com/apitoolkit/apitoolkit-go"
)

func main() {
	ctx := context.Background()
	apitoolkitClient, err := apitoolkit.NewClient(ctx, apitoolkit.Config{APIKey: "<API_KEY>"})
	if err != nil {
		panic(err)
	}

	helloHandler := func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("non-existing-file.txt")
		if err!= nil {
			// Report the error to apitoolkit
			apitoolkit.ReportError(r.Context(), err)
		}
		fmt.Fprintln(w, "{\"Hello\": \"World!\"}")
	}

	http.Handle("/", apitoolkitClient.Middleware(http.HandlerFunc(helloHandler)))

	if err := http.ListenAndServe(":8089", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}

```

**Gin**

```go
package main

import (
    "github.com/gin-gonic/gin"
  	apitoolkit "github.com/apitoolkit/apitoolkit-go"
)

func main() {
    r := gin.Default()
	apitoolkitClient, err := apitoolkit.NewClient(context.Background(), apitoolkit.Config{APIKey: "<APIKEY>"})
	if err != nil {
    	panic(err)
	}

    r.Use(apitoolkitClient.GinMiddleware)

    r.GET("/", func(c *gin.Context) {
		file, err := os.Open("non-existing-file.txt")
		if err!= nil {
			// Report an error to apitoolkit
			apitoolkit.ReportError(c.Request.Context(), err)
		}
        c.String(http.StatusOK, "Hello, World!")
    })

    r.Run(":8080")
}
```

**Echo**

```go
package main

import (
   //... other imports
  	apitoolkit "github.com/apitoolkit/apitoolkit-go"
)

func main() {
	e := echo.New()
	ctx := context.Background()

	apitoolkitClient, err := apitoolkit.NewClient(ctx, apitoolkit.Config{APIKey: "<API_KEY>"})
	if err != nil {
		panic(err)
	}

	e.Use(apitoolkitClient.EchoMiddleware)

	e.GET("/", hello)

	e.Logger.Fatal(e.Start(":1323"))
}

func hello(c echo.Context) error {
	file, err := os.Open("non-existing-file.txt")
	if err != nil {
		apitoolkit.ReportError(c.Request().Context(), err)
	}
	log.Println(file)
	return c.String(http.StatusOK, "Hello, World!")
}

```

**Gorilla mux**

```go
import (
   //... other imports
  	apitoolkit "github.com/apitoolkit/apitoolkit-go"
)

func main() {
	r := mux.NewRouter()
	ctx := context.Background()

	apitoolkitClient, err := apitoolkit.NewClient(ctx, apitoolkit.Config{APIKey: "<API_KEY>"})
	if err != nil {
		panic(err)
	}
	r.Use(apitoolkitClient.GorillaMuxMiddleware)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := os.Open("mux.json")
		if err != nil {
			// Report the error to apitoolkit
			apitoolkit.ReportError(r.Context(), err)
		}
		fmt.Fprintln(w, "Hello, World!")
	})

	server := &http.Server{Addr: ":8080", Handler: r}
	err = server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}
}

```
