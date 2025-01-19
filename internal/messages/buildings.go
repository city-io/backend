package messages

import (
	"cityio/internal/models"

	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

type CreateBuildingMessage struct {
	Building models.Building
	Restore  bool
}
type UpdateBuildingTilePIDMessage struct {
	TilePID *actor.PID
}
type GetBuildingMessage struct{}
type DeleteBuildingMessage struct {
	BuildingId string
}

type CreateBuildingResponseMessage struct {
	Error error
}
type GetBuildingResponseMessage struct {
	Building models.Building
}
type DeleteBuildingResponseMessage struct {
	Error error
}

type BuildingTypeNotFoundError struct {
	BuildingType string
}

func (e *BuildingTypeNotFoundError) Error() string {
	return fmt.Sprintf("Building type not found: %s", e.BuildingType)
}
