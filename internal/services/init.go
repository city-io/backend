package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
)

var db = database.GetDb()
var system = actors.GetSystem()

func Init() {
	managerPID := actors.GetManagerPID()
	initResponse, err := actors.Request[messages.InitPIDManagerResponseMessage](system.Root, managerPID, messages.InitPIDManagerMessage{})
	if err != nil {
		panic(err)
	}
	if initResponse.Error != nil {
		panic(initResponse.Error)
	}

	var users []models.User
	db.Find(&users)

	for _, user := range users {
		err := RestoreUser(user)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Spawned actors for %d users", len(users))

	var mapTiles []models.MapTile
	db.Find(&mapTiles)

	for _, mapTile := range mapTiles {
		err := RestoreMapTile(mapTile)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Spawned actors for %d map tiles", len(mapTiles))

	var cities []models.City
	db.Find(&cities)

	for _, city := range cities {
		err := RestoreCity(city)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Spawned actors for %d cities", len(cities))

	var armies []models.Army
	db.Find(&armies)

	for _, army := range armies {
		err := RestoreArmy(army)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Spawned actors for %d armies", len(armies))

	var buildings []models.Building
	db.Find(&buildings)

	for _, building := range buildings {
		err := RestoreBuilding(building)
		if err != nil {
			panic(err)
		}
	}
}
