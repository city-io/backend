package messages

import (
	"cityio/internal/models"

	"fmt"
)

type CreateMapTileMessage struct {
	Tile    models.MapTile
	Restore bool
}
type GetMapTileMessage struct{}

type CreateMapTileResponseMessage struct {
	Error error
}
type GetMapTileResponseMessage struct {
	Tile models.MapTile
}

// Errors
type MapTileNotFoundError struct {
	X int
	Y int
}

func (e *MapTileNotFoundError) Error() string {
	return fmt.Sprintf("Map tile not found: %d,%d", e.X, e.Y)
}
