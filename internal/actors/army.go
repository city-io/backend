package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
)

type ArmyActor struct {
	BaseActor
	Army models.Army

	OwnerPID *actor.PID
}

func (state *ArmyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateArmyMessage:
		state.Army = msg.Army
		state.OwnerPID = msg.OwnerPID

		if !msg.Restore {
			ctx.Send(state.database, messages.CreateArmyMessage{
				Army: state.Army,
			})
		}
		ctx.Respond(messages.CreateArmyResponseMessage{
			Error: nil,
		})

	case messages.GetArmyMessage:
		ctx.Respond(messages.GetArmyResponseMessage{
			Army: state.Army,
		})

	case messages.DeleteArmyMessage:
		ctx.Send(state.database, messages.DeleteArmyMessage{
			ArmyId: state.Army.ArmyId,
		})
		ctx.Respond(messages.DeleteArmyResponseMessage{
			Error: nil,
		})
		log.Printf("Shutting down ArmyActor for army: %s", state.Army.ArmyId)
		ctx.Stop(ctx.Self())
	}
}

func (state *ArmyActor) getTilePID() (*actor.PID, error) {
	getTilePIDResponse, err := Request[messages.GetMapTilePIDResponseMessage](system.Root, GetManagerPID(), messages.GetMapTilePIDMessage{
		X: state.Army.TileX,
		Y: state.Army.TileY,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return nil, err
	}
	if getTilePIDResponse.PID == nil {
		log.Printf("Error restoring army: Map tile not found")
		return nil, &messages.MapTileNotFoundError{X: state.Army.TileX, Y: state.Army.TileY}
	}
	return getTilePIDResponse.PID, nil
}
