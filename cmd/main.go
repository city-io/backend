package main

import (
	"cityio/internal/models"
	"cityio/internal/services"
)

func main() {
	services.Init()
	services.RegisterUser(models.UserInput{
		Email:    "test@gmail.com",
		Username: "test",
		Password: "test",
	})
}
