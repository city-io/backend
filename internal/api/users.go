package api

import (
	"log"
	"net/http"
)

func Register(response http.ResponseWriter, request *http.Request) {
	log.Println("Received POST /users/register")
}
