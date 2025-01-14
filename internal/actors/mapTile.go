package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
)

type MapTileActor struct {
	BaseActor
	Tile        models.MapTile
	CityPID     *actor.PID
	BuildingPID *actor.PID
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

	case messages.UpdateTileBuildingPIDMessage:
		state.BuildingPID = msg.BuildingPID

	case messages.GetMapTileMessage:
		var city *models.City = nil
		if state.CityPID != nil {
			response, err := Request[messages.GetCityResponseMessage](ctx, state.CityPID, messages.GetCityMessage{})
			if err != nil {
				log.Printf("Error getting city: %s", err)
			} else {
				city = &response.City
			}
		}
		var building *models.Building = nil
		if state.BuildingPID != nil {
			response, err := Request[messages.GetBuildingResponseMessage](ctx, state.BuildingPID, messages.GetBuildingMessage{})
			if err != nil {
				log.Printf("Error getting building: %s", err)
			} else {
				building = &response.Building
			}
		}
		ctx.Respond(messages.GetMapTileResponseMessage{
			Tile:     state.Tile,
			City:     city,
			Building: building,
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
