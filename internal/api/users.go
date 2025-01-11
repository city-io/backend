package api

import (
	"cityio/internal/constants"
	"cityio/internal/models"
	"cityio/internal/services"

	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

func Register(response http.ResponseWriter, request *http.Request) {
	log.Println("Received POST /users/register")

	user, err := DecodeBody[models.UserInput](request)
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

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	_, err = services.CreateCity(models.City{
		Type:       "city",
		Owner:      userId,
		Name:       fmt.Sprintf("%s's City", user.Username),
		Population: constants.INITIAL_PLAYER_CITY_POPULATION,
		StartX:     r.Intn(constants.MAP_SIZE - constants.CITY_SIZE),
		StartY:     r.Intn(constants.MAP_SIZE - constants.CITY_SIZE),
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
