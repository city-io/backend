package messages

import (
	"cityio/internal/models"

	"fmt"
)

type CreateCityMessage struct {
	City    models.City
	Restore bool
}
type GetCityMessage struct{}

type CreateCityResponseMessage struct {
	Error error
}
type GetCityResponseMessage struct {
	City models.City
}

// Errors
type CityNotFoundError struct {
	CityId string
}

func (e *CityNotFoundError) Error() string {
	return fmt.Sprintf("City not found: %s", e.CityId)
}
