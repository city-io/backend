package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/models"

	"log"
)

var db = database.GetDb()
var system = actors.GetSystem()

func Init() {
	var users []models.User
	db.Find(&users)

	for _, user := range users {
		RestoreUser(user)
	}
	log.Printf("Spawned actors for %d users", len(users))

	var mapTiles []models.MapTile
	db.Find(&mapTiles)

	for _, mapTile := range mapTiles {
		RestoreMapTile(mapTile)
	}
	log.Printf("Spawned actors for %d map tiles", len(mapTiles))

	var cities []models.City
	db.Find(&cities)

	for _, city := range cities {
		err := RestoreCity(city)
		if err != nil {
			log.Printf("Failed to restore city %s: %s", city.CityId, err.Error())
			panic(err)
		}
	}
	log.Printf("Spawned actors for %d cities", len(cities))
}
