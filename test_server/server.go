package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)


func InitiateServer(res http.ResponseWriter, req *http.Request) {
	// for possible authentication check...though I dont know what the auth design will be like yet...implementing this might even pose a security risk plus it'll violate the no env files rule..just doing this to notify you about the possibility of auth actions

	// service_id := mux.Vars(req)["service_id"]
	// service_pin := mux.Vars(req)["service_pin"]

	
	// if service_id == "" {
	// 	if service_pin != "" {
	// 	}
	// }

	credFile, err := os.Open("path to credential.json")
	if err != nil {
		fmt.Println(err)
	}

	defer credFile.Close()

	file, _ := ioutil.ReadAll(credFile)
	var data map[string]interface{}
	json.Unmarshal([]byte(file), &data)

	if err := json.NewEncoder(res).Encode(data); err != nil {
		fmt.Printf("error sending response: %v", err)
	}
}

func RunServer() {
	router := mux.NewRouter().StrictSlash(true)

	// for possible authentication 
	// router.HandleFunc("/testserver/{service_id}/{service_pin}", InitiateServer)

	router.HandleFunc("/testserver", InitiateServer)

	log.Fatal(http.ListenAndServe(":8000", router))
}