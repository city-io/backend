package api

import (
	"cityio/internal/models"
	"cityio/internal/services"

	"log"
	"net/http"
)

func Register(response http.ResponseWriter, request *http.Request) {
	log.Println("Received POST /users/register")

	user, err := DecodeBody[models.UserInput](request)
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = services.RegisterUser(user)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.WriteHeader(http.StatusOK)
}
