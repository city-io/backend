package ports

import "cityio/internal/models"

type Controllers interface {
	User() UserController
	City() CityController
	Tile() TileController
}

type UserController interface {
	Restore(user *models.User) error
	Create(user *models.CreateUserRequest) (string, error)
}

type CityController interface {
	Restore(city *models.City) error
	Create(city *models.CityInput) (*models.City, error)
}

type TileController interface {
	Restore(tile *models.Tile) error
	Create(city *models.Tile) error
}
