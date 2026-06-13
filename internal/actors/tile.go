package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/messages"
)

type tileActor struct {
	baseActor

	CityID     *string
	BuildingID *string
}

func NewTileActor() BaseActorInterface {
	return &tileActor{}
}

func (*tileActor) ActorType() string {
	return "tile"
}

func (state *tileActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.UpdateTileCityMessage:
		state.CityID = &msg.CityID
		ctx.Respond(messages.Ack{})

	case messages.UpdateTileBuildingMessage:
		state.BuildingID = msg.BuildingID
		ctx.Respond(messages.Ack{})

	case messages.GetTileMessage:
		ctx.Respond(messages.GetTileResponseMessage{
			CityID:     state.CityID,
			BuildingID: state.BuildingID,
		})
	}
}
