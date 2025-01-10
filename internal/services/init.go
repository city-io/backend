package services

import (
	"cityio/internal/database"
	"cityio/internal/models"

	"github.com/asynkron/protoactor-go/actor"
)

var db = database.GetDb()

func Init(system *actor.ActorSystem) {
	var users []models.User
	db.Find(&users)

	for _, user := range users {
		RestoreUser(system, user)
	}
}
