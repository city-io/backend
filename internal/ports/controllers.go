package ports

import "cityio/internal/models"

type Controllers interface {
	User() UserController
	City() CityController
	Tile() TileController
	Building() BuildingController
}

type UserController interface {
	Restore(user *models.User) error
	Create(user *models.CreateUserRequest) (string, error)
}

type CityController interface {
	Restore(city *models.City) error
	Create(city *models.CityInput) (*models.City, error)
}

type TileController interface{}

type BuildingController interface {
	Restore(building *models.Building) error
	Create(building *models.Building) (*models.Building, error)
}
