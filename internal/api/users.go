package api

import (
	"cityio/internal/models"
	"cityio/internal/services"

	"encoding/json"
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
		json.NewEncoder(response).Encode(err.Error())
		return
	}

	response.WriteHeader(http.StatusOK)
}

func Login(response http.ResponseWriter, request *http.Request) {
	log.Println("Received POST /users/login")

	user, err := DecodeBody[models.UserInput](request)
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := services.LoginUser(user)
	if err != nil {
		response.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(response).Encode(err.Error())
		return
	}

	json.NewEncoder(response).Encode(token)
}
