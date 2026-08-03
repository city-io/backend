package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/messages"
)

type tileActor struct {
	baseActor

	CityID     *string
	BuildingID *string
	Armies     map[string]struct{}
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
		if ctx.Sender() != nil {
			ctx.Respond(messages.Ack{})
		}

	case messages.UpdateTileBuildingMessage:
		state.BuildingID = msg.BuildingID
		if ctx.Sender() != nil {
			ctx.Respond(messages.Ack{})
		}

	case messages.AddTileArmyMessage:
		if state.Armies == nil {
			state.Armies = make(map[string]struct{})
		}
		state.Armies[msg.ArmyID] = struct{}{}
		if ctx.Sender() != nil {
			ctx.Respond(messages.Ack{})
		}

	case messages.RemoveTileArmyMessage:
		delete(state.Armies, msg.ArmyID)
		if ctx.Sender() != nil {
			ctx.Respond(messages.Ack{})
		}

	case messages.GetTileMessage:
		armyIDs := make([]string, 0, len(state.Armies))
		for id := range state.Armies {
			armyIDs = append(armyIDs, id)
		}
		ctx.Respond(messages.GetTileResponseMessage{
			CityID:     state.CityID,
			BuildingID: state.BuildingID,
			ArmyIDs:    armyIDs,
		})
	}
}
