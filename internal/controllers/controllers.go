// Package controllers initializes the controllers for handling various application functionalities.
package controllers

import (
	"cityio/internal/logger"
	"cityio/internal/ports"
)

type Controllers struct {
	User     *UserController
	City     *CityController
	Tile     *TileController
	Building *BuildingController
}

func NewControllers(cp ports.ClusterProvider, l logger.Logger) *Controllers {
	return &Controllers{
		User:     NewUserController(cp, l),
		City:     NewCityController(cp, l),
		Tile:     NewTileController(cp, l),
		Building: NewBuildingController(cp, l),
	}
}
