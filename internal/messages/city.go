package messages

import (
	"fmt"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/models"
)

type CreateCityMessage struct {
	City     models.City
	TilePIDs map[int]map[int]*actor.PID
	OwnerPID *actor.PID
	Restore  bool
}

type UpdateCityMessage struct {
	City models.City
}
type UpdateCityOwnerMessage struct {
	Owner string
}
type UpdateCityPopulationCapMessage struct {
	Change float64
}

type GetCityMessage struct{}
type GetCityResponseMessage struct {
	City models.City
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
