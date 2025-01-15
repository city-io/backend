package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
)

type UserActor struct {
	BaseActor
	User     models.User
	ArmyPIDs map[string]*actor.PID
}

func (state *UserActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.RegisterUserMessage:
		state.User = msg.User
		state.ArmyPIDs = make(map[string]*actor.PID)
		if !msg.Restore {
			err := state.createUser()
			ctx.Respond(messages.RegisterUserResponseMessage{
				Error: err,
			})
		} else {
			ctx.Respond(messages.RegisterUserResponseMessage{
				Error: nil,
			})
		}

	case messages.UpdateUserGoldMessage:
		log.Printf("Changing %s's gold by: %d", state.User.Username, msg.Change)
		state.User.Gold += msg.Change
		ctx.Respond(messages.UpdateUserGoldResponseMessage{
			Error: nil,
		})

	case messages.UpdateUserFoodMessage:
		log.Printf("Changing %s's food by: %d", state.User.Username, msg.Change)
		state.User.Food += msg.Change
		ctx.Respond(messages.UpdateUserFoodResponseMessage{
			Error: nil,
		})

	case messages.GetUserMessage:
		ctx.Respond(messages.GetUserResponseMessage{
			User: state.User,
		})

	case messages.AddUserArmyMessage:
		state.ArmyPIDs[msg.ArmyId] = msg.ArmyPID
		ctx.Respond(messages.AddUserArmyResponseMessage{
			Error: nil,
		})

	case messages.DeleteUserMessage:
		result := state.db.Delete(&state.User)
		ctx.Respond(messages.DeleteUserResponseMessage{
			Error: result.Error,
		})
		log.Printf("Shutting down UserActor for user: %s", state.User.Username)
		ctx.Stop(ctx.Self())
	}
}

func (state *UserActor) createUser() error {
	result := state.db.Create(&state.User)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
