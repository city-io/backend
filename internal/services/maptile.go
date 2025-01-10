package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/state"

	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

func RestoreMapTile(tile models.MapTile) {
	log.Printf("Restoring map tile at: %d,%d", tile.X, tile.Y)
	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewMapTileActor(database.GetDb())
	})
	newPID := system.Root.Spawn(props)
	system.Root.Send(newPID, messages.CreateMapTileMessage{
		Tile:    tile,
		Restore: true,
	})
	state.AddMapTilePID(tile.X, tile.Y, newPID)
}

func CreateMapTile(tile models.MapTile) error {
	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewMapTileActor(database.GetDb())
	})
	newPID := system.Root.Spawn(props)
	future := system.Root.RequestFuture(newPID, messages.CreateMapTileMessage{
		Tile:    tile,
		Restore: false,
	}, time.Second*2)

	response, err := future.Result()
	if err != nil {
		return err
	}

	if response, ok := response.(messages.CreateMapTileResponseMessage); ok {
		if response.Error != nil {
			return response.Error
		}
	} else {
		return &messages.InternalError{}
	}

	state.AddMapTilePID(tile.X, tile.Y, newPID)
	return nil
}

func GetMapTile(x int, y int) (models.MapTile, error) {
	tilePID, exists := state.GetMapTilePID(x, y)
	if !exists {
		return models.MapTile{}, &messages.MapTileNotFoundError{X: x, Y: y}
	}

	future := system.Root.RequestFuture(tilePID, messages.GetMapTileMessage{}, time.Second*2)
	response, err := future.Result()
	if err != nil {
		return models.MapTile{}, err
	}

	tile, ok := response.(models.MapTile)
	if !ok {
		return models.MapTile{}, &messages.MapTileNotFoundError{X: x, Y: y}
	}

	return tile, nil
}
