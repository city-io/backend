package main

import (
	"cityio/internal/database"

	"log"
	// "github.com/asynkron/protoactor-go/actor"
)

func main() {
	database.GetDb()
	log.Println("Initiated actor system.")
}
