package api

import (
	"cityio/internal/constants"
	"cityio/internal/models"
	"cityio/internal/services"

	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func Register(response http.ResponseWriter, request *http.Request) {
	log.Println("Received POST /users/register")

	user, err := DecodeBody[models.RegisterUserRequest](request)
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	var userId string
	userId, err = services.RegisterUser(user)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(response).Encode(err.Error())
		return
	}

	_, err = services.CreateCity(models.CityInput{
		Type:       "city",
		Owner:      userId,
		Name:       fmt.Sprintf("%s's City", user.Username),
		Population: constants.INITIAL_PLAYER_CITY_POPULATION,
		Size:       constants.CITY_SIZE,
	})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(response).Encode(err.Error())
		return
	}

	response.WriteHeader(http.StatusOK)
}

func Login(response http.ResponseWriter, request *http.Request) {
	log.Println("Received POST /users/login")

	user, err := DecodeBody[models.LoginUserRequest](request)
	if err != nil || user.Identifier == "" || user.Password == "" {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	loginResponse, err := services.LoginUser(user)
	if err != nil {
		response.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(response).Encode(err.Error())
		return
	}

	json.NewEncoder(response).Encode(loginResponse)
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

func DeleteUser(response http.ResponseWriter, request *http.Request) {
	log.Println("Received DELETE /users/delete")

	vars := mux.Vars(request)
	userId := vars["userId"]

	err := services.DeleteUserCity(userId)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(response).Encode(err.Error())
		return
	}

	err = services.DeleteUser(userId)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(response).Encode(err.Error())
		return
	}

	response.WriteHeader(http.StatusOK)
}
