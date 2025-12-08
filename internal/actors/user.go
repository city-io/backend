package actors

import (
	"context"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/logger"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/services"
	"cityio/internal/ws"
)

type userActor struct {
	baseActor
	User models.User

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func NewUserActor() BaseActorInterface {
	return &userActor{}
}

func (state *userActor) ActorType() string {
	return "user"
}

func (state *userActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case *messages.CreateUserMessage:
		state.Log.Info("registering UserActor", "username", msg.User.Username)
		state.User = msg.User
		if !msg.Restore {
			ctx.Send(state.Cluster.DB(), &messages.CreateUserMessage{
				User: state.User,
			})
			serviceCtx := logger.WithContext(context.Background(), state.Log)
			services.CreateCity(serviceCtx, state.Cluster, &models.CityInput{
				Type:  constants.CityTypeCity,
				Owner: &state.User.UserID,
				Name:  fmt.Sprintf("%s's City", state.User.Username),
				Size:  constants.CitySize,
			})
		}
		state.startPeriodicOperation(ctx)
		ctx.Respond(messages.Ack{})

	case messages.UpdateUserGoldMessage:
		state.User.Gold += msg.Change
		state.ws()

	case messages.UpdateUserFoodMessage:
		state.User.Food += msg.Change
		state.ws()

	case messages.CheckAndDeductGoldMessage:
		if missing := msg.Amount - state.User.Gold; missing > 0 {
			ctx.Respond(messages.InsufficientGoldError{
				Missing: missing,
			})
		}
		state.ws()
		ctx.Respond(messages.Ack{})

	case messages.GetUserMessage:
		ctx.Respond(&messages.GetUserResponseMessage{
			User: state.User,
		})

	case messages.DeleteUserMessage:
		ctx.Send(state.Cluster.DB(), messages.DeleteUserMessage{
			UserID: state.User.UserID,
		})

		state.Log.Info("shutting down UserActor", "user_id", state.User.UserID)
		state.stopPeriodicOperation()
		ctx.Stop(ctx.Self())

	case messages.PeriodicOperationMessage:
		// make a backup of the user state
		ctx.Send(state.Cluster.DB(), &messages.UpdateUserMessage{
			User: state.User,
		})
	}
}

func (state *userActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.UserBackupFrequency * time.Second)
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

func (state *userActor) stopPeriodicOperation() {
	close(state.stopTickerCh)
	state.ticker = nil
}

func (state *userActor) ws() {
	ws.Send(state.User.UserID, messages.WS_USER, &models.UserAccountOutput{
		Username: state.User.Username,
		Gold:     state.User.Gold,
		Food:     state.User.Food,
	})
}
