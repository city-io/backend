package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

type MapTileActor struct {
	BaseActor
	Tile    models.MapTile
	CityPID *actor.PID
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
		} else {
			ctx.Respond(messages.CreateMapTileResponseMessage{
				Error: nil,
			})
		}

	case messages.UpdateTileCityPIDMessage:
		state.CityPID = msg.CityPID

	case messages.GetMapTileMessage:
		var city *models.City = nil
		if state.CityPID != nil {
			future := ctx.RequestFuture(state.CityPID, messages.GetCityMessage{}, time.Second*2)
			response, err := future.Result()
			if err != nil {
				log.Printf("Error getting city for tile: %s", err)
			}
			cityValue := response.(messages.GetCityResponseMessage).City
			city = &cityValue
		}
		ctx.Respond(messages.GetMapTileResponseMessage{
			Tile: state.Tile,
			City: city,
		})
	}
}

func (state *MapTileActor) createMapTile() error {
	result := state.db.Create(&state.Tile)
	if result.Error != nil {
		log.Printf("Error creating map tile: %s", result.Error)
		return result.Error
	}
	return nil
}
