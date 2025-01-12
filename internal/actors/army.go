package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
	"gorm.io/gorm"
)

type ArmyActor struct {
	Db       *gorm.DB
	Army     models.Army
	OwnerPID *actor.PID
}

func NewArmyActor(db *gorm.DB) *ArmyActor {
	actor := &ArmyActor{
		Db: db,
	}
	return actor
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
		}

	case messages.GetArmyMessage:
		ctx.Respond(messages.GetArmyResponseMessage{
			Army: state.Army,
		})

	case messages.DeleteArmyMessage:
		result := state.Db.Delete(&state.Army)
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
	result := state.Db.Create(&state.Army)
	if result.Error != nil {
		log.Printf("Error creating army: %s", result.Error)
		return result.Error
	}
	return nil
}
