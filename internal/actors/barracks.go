package actors

import (
	"cityio/internal/messages"

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
			state.createBuilding(ctx)
		}
		ctx.Respond(messages.CreateBuildingResponseMessage{
			Error: nil,
		})

	case messages.UpdateBuildingTilePIDMessage:
		state.MapTilePID = msg.TilePID

	case messages.GetBuildingMessage:
		state.getBuilding(ctx)

	case messages.DeleteBuildingMessage:
		state.deleteBuilding(ctx)
	}
}
