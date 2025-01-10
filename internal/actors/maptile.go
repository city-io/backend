package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
	"gorm.io/gorm"
)

type MapTileActor struct {
	Db   *gorm.DB
	Tile models.MapTile
}

func NewMapTileActor(db *gorm.DB) *MapTileActor {
	actor := &MapTileActor{
		Tile: models.MapTile{},
		Db:   db,
	}
	return actor
}

func (state *MapTileActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateMapTileMessage:
		state.Tile = msg.Tile
		if !msg.Restore {
			err := state.createMapTile()
			ctx.Respond(messages.CreateMapTileResponseMessage{
				Error: err,
			})
		}

	case messages.GetMapTileMessage:
		ctx.Respond(messages.GetMapTileResponseMessage{
			Tile: state.Tile,
		})
	}
}

func (state *MapTileActor) createMapTile() error {
	result := state.Db.Create(&state.Tile)
	if result.Error != nil {
		log.Printf("Error creating map tile: %s", result.Error)
		return result.Error
	}
	return nil
}
