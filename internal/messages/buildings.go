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

type CreateBuildingResponseMessage struct {
	Error error
}

type GetBuildingMessage struct{}
type GetBuildingResponseMessage struct {
	Building models.Building
}

type DeleteBuildingMessage struct{}
type DeleteBuildingResponseMessage struct {
	Error error
}

type BuildingTypeNotFoundError struct {
	BuildingType string
}

func (e *BuildingTypeNotFoundError) Error() string {
	return fmt.Sprintf("Building type not found: %s", e.BuildingType)
}
