package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/messages"
	"cityio/internal/ports"
)

type tileActor struct {
	baseActor

	CityID     *string
	BuildingID *string
}

func NewTileActor() ports.BaseActorInterface {
	return &tileActor{}
}

func (*tileActor) ActorType() string {
	return "tile"
}

func (state *tileActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.UpdateTileOwnerMessage:
		if state.BuildingID != nil {
			state.Cluster.Tell("building", *state.BuildingID, messages.UpdateBuildingOwnerMessage(msg))
		}
		ctx.Respond(messages.Ack{})

	case messages.UpdateTileCityMessage:
		state.CityID = &msg.CityID
		ctx.Respond(messages.Ack{})

	case messages.UpdateTileBuildingMessage:
		state.BuildingID = msg.BuildingID
		ctx.Respond(messages.Ack{})

	case messages.GetTileMessage:
		ctx.Respond(messages.GetTileResponseMessage{
			City: nil,
		})
	}
}
