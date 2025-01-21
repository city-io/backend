package actors

import (
	// "cityio/internal/constants"
	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

type UserActor struct {
	BaseActor
	User     models.User
	ArmyPIDs map[string]*actor.PID

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func (state *UserActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.RegisterUserMessage:
		state.User = msg.User
		state.ArmyPIDs = make(map[string]*actor.PID)
		if !msg.Restore {
			ctx.Send(state.database, messages.RegisterUserMessage{
				User: state.User,
			})
		}
		ctx.Respond(messages.RegisterUserResponseMessage{
			Error: nil,
		})
		state.startPeriodicOperation(ctx)

	case messages.UpdateUserGoldMessage:
		// log.Printf("Changing %s's gold by: %d", state.User.Username, msg.Change)
		state.User.Gold += msg.Change
		ctx.Respond(messages.UpdateUserGoldResponseMessage{
			Error: nil,
		})

	case messages.UpdateUserFoodMessage:
		// log.Printf("Changing %s's food by: %d", state.User.Username, msg.Change)
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
		ctx.Send(state.database, messages.DeleteUserMessage{
			UserId: state.User.UserId,
		})
		ctx.Respond(messages.DeleteUserResponseMessage{
			Error: nil,
		})
		log.Printf("Shutting down UserActor for user: %s", state.User.Username)
		state.stopPeriodicOperation()
		ctx.Stop(ctx.Self())

	case messages.PeriodicOperationMessage:
		// make a backup of the user state
		ctx.Send(state.database, &messages.UpdateUserMessage{
			User: state.User,
		})
	}
}

func (state *UserActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.USER_BACKUP_FREQUENCY * time.Second)
	state.stopTickerCh = make(chan struct{})

	go func() {
		for {
			select {
			case <-state.ticker.C:
				ctx.Send(ctx.Self(), messages.PeriodicOperationMessage{})
			case <-state.stopTickerCh:
				state.ticker.Stop()
				return
			}
		}
	}()
}

func (state *UserActor) stopPeriodicOperation() {
	close(state.stopTickerCh)
	state.ticker = nil
}
