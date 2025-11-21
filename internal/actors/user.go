package actors

import (
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
	"cityio/internal/ws"
)

type UserActor struct {
	models.BaseActor
	User     models.User
	ArmyPIDs map[string]*actor.PID

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func NewUserActor() ports.BaseActorInterface {
	return &UserActor{}
}

func (state *UserActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.RegisterUserMessage:
		state.User = msg.User
		state.ArmyPIDs = make(map[string]*actor.PID)
		if !msg.Restore {
			ctx.Send(state.Database, messages.RegisterUserMessage{
				User: state.User,
			})
		}
		state.startPeriodicOperation(ctx)

	case messages.AddAllyMessage:
		state.User.Allies = append(state.User.Allies, msg.Ally)
		state.ws()
		ctx.Respond(messages.AddAllyResponseMessage{
			Error: nil,
		})

	case messages.RemoveAllyMessage:
		for i, ally := range state.User.Allies {
			if ally == msg.Ally {
				state.User.Allies = append(state.User.Allies[:i], state.User.Allies[i+1:]...)
				break
			}
		}
		state.ws()
		ctx.Respond(messages.RemoveAllyResponseMessage{
			Error: nil,
		})

	case messages.UpdateUserGoldMessage:
		state.User.Gold += msg.Change
		state.ws()

		ctx.Respond(messages.UpdateUserGoldResponseMessage{
			Error: nil,
		})

	case messages.UpdateUserFoodMessage:
		state.User.Food += msg.Change
		state.ws()
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
		ctx.Send(state.Database, messages.DeleteUserMessage{
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
		ctx.Send(state.Database, &messages.UpdateUserMessage{
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

func (state *UserActor) ws() {
	ws.Send(state.User.UserId, messages.WS_USER, &models.UserAccountOutput{
		Username: state.User.Username,
		Gold:     state.User.Gold,
		Food:     state.User.Food,
		Allies:   state.User.Allies,
	})
}
