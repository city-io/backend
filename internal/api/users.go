package api

import (
	"cityio/internal/models"
	"cityio/internal/services"

	"encoding/json"
	"log"
	"net/http"
	"strings"
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

	type LoginResponse struct {
		Token string `json:"token"`
	}

	json.NewEncoder(response).Encode(LoginResponse{
		Token: token,
	})
}

func ValidateToken(response http.ResponseWriter, request *http.Request) {
	log.Println("Received GET /users/validate")

	token := strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		response.WriteHeader(http.StatusUnauthorized)
		return
	}

	claims, err := services.ValidateToken(token)
	if err != nil {
		response.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(response).Encode(err.Error())
		return
	}

	json.NewEncoder(response).Encode(claims)
}
