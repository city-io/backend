package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/models"
)

var db = database.GetDb()
var system = actors.GetSystem()

func Init() {
	var users []models.User
	db.Find(&users)

	for _, user := range users {
		RestoreUser(user)
	}

	// var cities []models.City

}
