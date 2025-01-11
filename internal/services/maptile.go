package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/state"

	"time"

	"github.com/asynkron/protoactor-go/actor"
)

func RestoreMapTile(tile models.MapTile) {
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

func GetMapTile(x int, y int) (models.MapTileOutput, error) {
	tilePID, exists := state.GetMapTilePID(x, y)
	if !exists {
		return models.MapTileOutput{}, &messages.MapTileNotFoundError{X: x, Y: y}
	}

	future := system.Root.RequestFuture(tilePID, messages.GetMapTileMessage{}, time.Second*2)
	result, err := future.Result()
	if err != nil {
		return models.MapTileOutput{}, err
	}

	response, ok := result.(messages.GetMapTileResponseMessage)
	if !ok {
		return models.MapTileOutput{}, &messages.MapTileNotFoundError{X: x, Y: y}
	}

	return models.MapTileOutput{
		X:    response.Tile.X,
		Y:    response.Tile.Y,
		City: response.City,
	}, nil
}
