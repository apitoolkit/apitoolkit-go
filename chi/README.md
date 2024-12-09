<div align="center">

![APItoolkit's Logo](https://github.com/apitoolkit/.github/blob/main/images/logo-white.svg?raw=true#gh-dark-mode-only)
![APItoolkit's Logo](https://github.com/apitoolkit/.github/blob/main/images/logo-black.svg?raw=true#gh-light-mode-only)

## Golang Chi SDK

[![APItoolkit SDK](https://img.shields.io/badge/APItoolkit-SDK-0068ff?logo=go)](https://github.com/topics/apitoolkit-sdk) [![Join Discord Server](https://img.shields.io/badge/Chat-Discord-7289da)](https://apitoolkit.io/discord?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme) [![APItoolkit Docs](https://img.shields.io/badge/Read-Docs-0068ff)](https://apitoolkit.io/docs/sdks/golang?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme) [![GoDoc](https://godoc.org/github.com/apitoolkit/apitoolkit-go?status.svg)](https://godoc.org/github.com/apitoolkit/apitoolkit-go/main/tree/chi)

APItoolkit is an end-to-end API and web services management toolkit for engineers and customer support teams. To integrate your Golang application with APItoolkit, you need to use this SDK to monitor incoming traffic, aggregate the requests, and then deliver them to the APItoolkit's servers.

</div>

---

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Contributing and Help](#contributing-and-help)
- [License](#license)

---

## Installation

Kindly run the command below to install the SDK:

```sh
go get github.com/apitoolkit/apitoolkit-go/chi
```

Then add `github.com/apitoolkit/apitoolkit-go/chi` to the list of dependencies like so:

```go
package main

import (
	"context"
	"log"
	"net/http"

	apitoolkitchi "github.com/apitoolkit/apitoolkit-go/chi"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)


func main() {
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("error shutting down tracer: %v", err)
		}
	}()

	tracer := otel.Tracer("example-chi-server")
	r := chi.NewRouter()

	// Add the apitoolkit chi middleware to monitor http requests
	// And report errors to apitoolkit
	r.Use(apitoolkitchi.Middleware(apitoolkitchi.Config{
		Debug:               false,
		ServiceName:         "example-chi-server",
		ServiceVersion:      "0.0.1",
		Tags:                []string{"env:dev"},
		CaptureRequestBody:  true,
		CaptureResponseBody: true,
		RedactHeaders:       []string{"Authorization", "X-Api-Key"},
		RedactRequestBody:   []string{"password", "credit_card"},
		RedactResponseBody:  []string{"password", "credit_card"},
	}))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})

	if err := http.ListenAndServe(":8000", r); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
```

> [!IMPORTANT]
>
> To learn more configuration options (redacting fields, error reporting, outgoing requests, etc.), please read this [SDK documentation](https://apitoolkit.io/docs/sdks/golang/chi/).

## Contributing and Help

To contribute to the development of this SDK or request help from the community and our team, kindly do any of the following:

- Read our [Contributors Guide](https://github.com/apitoolkit/.github/blob/main/CONTRIBUTING.md).
- Join our community [Discord Server](https://apitoolkit.io/discord?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme).
- Create a [new issue](https://github.com/apitoolkit/apitoolkit-go/issues/new/choose) in this repository.

## License

This repository is published under the [MIT](LICENSE) license.

---

<div align="center">

<a href="https://apitoolkit.io?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme" target="_blank" rel="noopener noreferrer"><img src="https://github.com/apitoolkit/.github/blob/main/images/icon.png?raw=true" width="40" /></a>

</div>
