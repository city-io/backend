package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
)

type ArmyActor struct {
	BaseActor
	Army     models.Army
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
