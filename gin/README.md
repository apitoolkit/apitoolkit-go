<div align="center">

![APItoolkit's Logo](https://github.com/apitoolkit/.github/blob/main/images/logo-white.svg?raw=true#gh-dark-mode-only)
![APItoolkit's Logo](https://github.com/apitoolkit/.github/blob/main/images/logo-black.svg?raw=true#gh-light-mode-only)

## Golang Gin SDK

[![APItoolkit SDK](https://img.shields.io/badge/APItoolkit-SDK-0068ff?logo=go)](https://github.com/topics/apitoolkit-sdk) [![Join Discord Server](https://img.shields.io/badge/Chat-Discord-7289da)](https://apitoolkit.io/discord?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme) [![APItoolkit Docs](https://img.shields.io/badge/Read-Docs-0068ff)](https://apitoolkit.io/docs/sdks/golang?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme) [![GoDoc](https://godoc.org/github.com/apitoolkit/apitoolkit-go?status.svg)](https://godoc.org/github.com/apitoolkit/apitoolkit-go/gin)

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
go get github.com/apitoolkit/apitoolkit-go/gin
```

Then add `github.com/apitoolkit/apitoolkit-go/gin` to the list of dependencies, like so:

```go
package main

import (
  apitoolkit "github.com/apitoolkit/apitoolkit-go/gin"
)
```

## Configuration

Next, initialize APItoolkit in your application's entry point (e.g., `main.go`) like so:

```go
package main

import (
  "context"
  "log"
  "net/http"
  "github.com/gin-gonic/gin"
  apitoolkit "github.com/apitoolkit/apitoolkit-go/gin"
)

func main() {
  ctx := context.Background()

  router := gin.New()
  // Initialize the client
  apitoolkitClient, err := apitoolkit.NewClient(ctx, apitoolkit.Config{APIKey: "{ENTER_YOUR_API_KEY_HERE}"})
  if err != nil {
    panic(err)
  }

  // Register APItoolkit's Gin middleware
  router.Use(apitoolkit.GinMiddleware(apitoolkitClient))
  router.GET("/", func(c *gin.Context) {
  	c.JSON(200, gin.H{
  		"message": "Hello World",
  	})
  })

  router.Run(":8080")
}
```

> [!IMPORTANT]
>
> To learn more configuration options (redacting fields, error reporting, outgoing requests, etc.), please read this [SDK documentation](https://apitoolkit.io/docs/sdks/golang/gin?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme).

## Contributing and Help

To contribute to the development of this SDK or request help from the community and our team, kindly do any of the following:

- Read our [Contributors Guide](https://github.com/apitoolkit/.github/blob/main/CONTRIBUTING.md).
- Join our community [Discord Server](https://apitoolkit.io/discord?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme).
- Create a [new issue](https://github.com/apitoolkit/apitoolkit-go/issues/new/choose) in this repository.

## License

This repository is published under the [MIT](../LICENSE) license.

---

<div align="center">
    
<a href="https://apitoolkit.io?utm_campaign=devrel&utm_medium=github&utm_source=sdks_readme" target="_blank" rel="noopener noreferrer"><img src="https://github.com/apitoolkit/.github/blob/main/images/icon.png?raw=true" width="40" /></a>

</div>
