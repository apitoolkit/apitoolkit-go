package conf

import (
	"log"

	"github.com/apitoolkit/apitoolkit-go-client/models"
)

// Configuration holds the receiver struct for methods
type Configuration struct {
	Log		*log.Logger
	Data	models.TestData
	// Pub 	*pubsub.Client
}