package api

import (
	"cityio/internal/services"

	"log"
	"net/http"
)

func ResetMap(response http.ResponseWriter, request *http.Request) {
	log.Println("Received POST /map/reset")

	go services.Reset()
	response.WriteHeader(http.StatusAccepted)
}
