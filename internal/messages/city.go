package messages

import (
	"fmt"

	"cityio/internal/domain"
)

type CreateCityMessage struct {
	City    domain.City
	Restore bool
}

type UpdateCityMessage struct {
	City domain.City
}
type UpdateCityOwnerMessage struct {
	Owner *string
}
type UpdateCityPopulationCapMessage struct {
	Change float64
}

type GetCityMessage struct{}
type GetCityResponseMessage struct {
	City domain.City
}

type DeleteCityMessage struct {
	CityID string
}

// Errors
type CityNotFoundError struct {
	CityId string
}

func (e *CityNotFoundError) Error() string {
	return fmt.Sprintf("City not found: %s", e.CityId)
}
