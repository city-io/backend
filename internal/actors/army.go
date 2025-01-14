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
			err := state.createArmy()
			ctx.Respond(messages.CreateArmyResponseMessage{
				Error: err,
			})
		} else {
			ctx.Respond(messages.CreateArmyResponseMessage{
				Error: nil,
			})
		}

	case messages.GetArmyMessage:
		ctx.Respond(messages.GetArmyResponseMessage{
			Army: state.Army,
		})

	case messages.DeleteArmyMessage:
		result := state.db.Delete(&state.Army)
		if result.Error != nil {
			log.Printf("Error deleting army: %s", result.Error)
		}
		ctx.Respond(messages.DeleteArmyResponseMessage{
			Error: result.Error,
		})
		log.Printf("Shutting down ArmyActor for army: %s", state.Army.ArmyId)
		ctx.Stop(ctx.Self())
	}
}

func (state *ArmyActor) createArmy() error {
	result := state.db.Create(&state.Army)
	if result.Error != nil {
		log.Printf("Error creating army: %s", result.Error)
		return result.Error
	}
	return nil
}
