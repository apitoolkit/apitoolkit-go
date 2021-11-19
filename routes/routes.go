package routes

import (
	"net/http"

	"github.com/apitoolkit/apitoolkit-go-client/handlers"
	"github.com/gorilla/mux"
)

func Routes() http.Handler {
	muX := mux.NewRouter()

	muX.HandleFunc("/create-topic/{project_id}", handlers.Repo.CreateTopic).Methods("POST")
	muX.HandleFunc("/publish-message", handlers.Repo.PublishMessage).Methods("POST")

	return muX
}