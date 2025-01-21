package messages

import (
	"cityio/internal/models"

	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

type CreateMapTileMessage struct {
	Tile    models.MapTile
	Restore bool
}
type AddTileArmyMessage struct {
	ArmyPID *actor.PID
	Army    models.Army
}
type GetMapTileMessage struct{}
type GetMapTileArmiesMessage struct{}

type CreateMapTileResponseMessage struct {
	Error error
}
type AddTileArmyResponseMessage struct {
	Error error
}
type GetMapTileResponseMessage struct {
	Tile     models.MapTile
	City     *models.City
	Building *models.Building
}
type GetMapTileArmiesResponseMessage struct {
	Armies []models.Army
}

// Errors
type MapTileNotFoundError struct {
	X int
	Y int
}

func (e *MapTileNotFoundError) Error() string {
	return fmt.Sprintf("Map tile not found: %d,%d", e.X, e.Y)
}
