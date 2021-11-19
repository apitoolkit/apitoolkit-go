package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Response struct {
	StatusCode	int			`json:"status_code"`
	Message		string		`json:"message"`
	Data		interface{}	`json:"data"`
} 

// ParseJSON parses the request body
func ParseJSON(req *http.Request, v interface{}) error {
	if req.Body == nil {
		return fmt.Errorf("empty request body")
	}

	decodedBody := json.NewDecoder(req.Body).Decode(v)

	return decodedBody
}


func Success(msg string, data interface{}, w http.ResponseWriter) {
	response := Response {
		Message: msg,
		StatusCode: http.StatusOK,
		Data: data,
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("error sending response: %v", err)
	}
}

func Error(statusCode int, w http.ResponseWriter) {
	response := Response {
		StatusCode: statusCode,
	}

	w.WriteHeader(response.StatusCode)
	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("error sending response: %v", err)
	}
}