package ports

import "cityio/internal/models"

type Controllers interface {
	User() UserController
	City() CityController
}

type UserController interface {
	RestoreUser(user *models.User) error
	RegisterUser(user *models.RegisterUserRequest) (string, error)
}

type CityController interface {
	RestoreCity(city *models.City) error
	CreateCity(city *models.CityInput) (*models.City, error)
}
