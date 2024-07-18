<div align="center">

![APItoolkit's Logo](https://github.com/apitoolkit/.github/blob/main/images/logo-white.svg?raw=true#gh-dark-mode-only)
![APItoolkit's Logo](https://github.com/apitoolkit/.github/blob/main/images/logo-black.svg?raw=true#gh-light-mode-only)

## Golang TLS Client SDK

[![APItoolkit SDK](https://img.shields.io/badge/APItoolkit-SDK-0068ff?logo=go)](https://github.com/topics/apitoolkit-sdk) [![Join Discord Server](https://img.shields.io/badge/Chat-Discord-7289da)](https://apitoolkit.io/discord?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme) [![APItoolkit Docs](https://img.shields.io/badge/Read-Docs-0068ff)](https://apitoolkit.io/docs/sdks/golang?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme) [![GoDoc](https://godoc.org/github.com/apitoolkit/apitoolkit-go?status.svg)](https://godoc.org/github.com/apitoolkit/apitoolkit-go/main/tree/native)

If you are using a TLS client for your HTTP requests, you will need to use the apitoolkit-go/tls_client package to monitor those requests. To use the package, you must first install it using the command below:

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
go get github.com/apitoolkit/apitoolkit-go/tls_client
```

Then add `github.com/apitoolkit/apitoolkit-go/tls_client` to the list of dependencies like so:

```go
package main

import (
  "context"
  "log"
  "net/http"

  fhttp "github.com/bogdanfinn/fhttp"
  tls_client "github.com/bogdanfinn/tls-client"

  apitoolkit "github.com/apitoolkit/apitoolkit-go/native"
  apitoolkitTlsClient "github.com/apitoolkit/apitoolkit-go/tls_client"
)

func main() {
  ctx := context.Background()

  apitoolkitClient, err := apitoolkit.NewClient(
    ctx,
    apitoolkit.Config{APIKey: "{ENTER_YOUR_API_KEY_HERE}"},
  )
  if err != nil {
    panic(err)
  }

  jar := tls_client.NewCookieJar()
  options := []tls_client.HttpClientOption{
    tls_client.WithTimeoutSeconds(30),
    tls_client.WithNotFollowRedirects(),
    tls_client.WithCookieJar(jar), // create cookieJar instance and pass it as argument
  }

  clientTLS, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
  if err != nil {
    panic(err)
  }

  http.HandleFunc("/test", apitoolkit.Middleware(apitoolkitClient)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

    tclient := apitoolkitTlsClient.NewHttpClient(r.Context(), clientTLS, apitoolkitClient)
    req, err := fhttp.NewRequest(http.MethodGet, "https://jsonplaceholder.typicode.com/posts/1", nil)
    if err != nil {
      panic(err)
    }

    resp, err := tclient.Do(req)
    if err != nil {
      panic(err)
    }

    log.Printf("status code: %d", resp.StatusCode)

    // Respond to the request
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Hello, World!"))
  })))

  http.ListenAndServe(":8080", nil)
}
```

> [!IMPORTANT]
>
> To learn more configuration options (redacting fields, error reporting, outgoing requests, etc.), please read this [SDK documentation](https://apitoolkit.io/docs/sdks/golang?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme).

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
