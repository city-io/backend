package messages

import (
	"cityio/internal/models"

	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

type CreateCityMessage struct {
	City     models.City
	TilePIDs map[int]map[int]*actor.PID
	Restore  bool
}
type GetCityMessage struct{}
type DeleteCityMessage struct {
	CityId string
}

type CreateCityResponseMessage struct {
	Error error
}
type GetCityResponseMessage struct {
	City models.City
}
type DeleteCityResponseMessage struct {
	Error error
}

// Errors
type CityNotFoundError struct {
	CityId string
}

func (e *CityNotFoundError) Error() string {
	return fmt.Sprintf("City not found: %s", e.CityId)
}
