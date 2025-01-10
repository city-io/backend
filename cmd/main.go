package main

import (
	"cityio/internal/models"
	"cityio/internal/services"

	"log"

	"github.com/asynkron/protoactor-go/actor"
)

func main() {
	system := actor.NewActorSystem()
	log.Println("Initiated actor system.")

	services.Init(system)
	services.RegisterUser(system, models.UserInput{
		Email:    "test@gmail.com",
		Username: "test",
		Password: "test",
	})
}
