package main

import (
	"log"
	"net/http"

	"github.com/apitoolkit/apitoolkit-go-client/routes"
)

const portNumber = ":8080"

func main() {
	srv := &http.Server {
		Addr: portNumber,
		Handler: routes.Routes(),
	}

	err := srv.ListenAndServe()

	log.Fatal(err)
}