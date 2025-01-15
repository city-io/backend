package actors

import (
	"cityio/internal/messages"

	"log"

	"github.com/asynkron/protoactor-go/actor"
)

type BarracksActor struct {
	BuildingActor
}

func (state *BarracksActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateBuildingMessage:
		state.Building = msg.Building

		if !msg.Restore {
			err := state.createBarracks()
			ctx.Respond(messages.CreateBuildingResponseMessage{
				Error: err,
			})
		} else {
			ctx.Respond(messages.CreateBuildingResponseMessage{
				Error: nil,
			})
		}

	case messages.UpdateBuildingTilePIDMessage:
		state.MapTilePID = msg.TilePID

	case messages.GetBuildingMessage:
		state.getBuilding(ctx)

	case messages.DeleteBuildingMessage:
		state.deleteBuilding(ctx)
	}
}

func (state *BarracksActor) createBarracks() error {
	result := state.db.Create(&state.Building)
	if result.Error != nil {
		log.Printf("Error creating barracks: %s", result.Error)
		return result.Error
	}
	return nil
}
