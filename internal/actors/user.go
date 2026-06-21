package actors

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/messages"
	"cityio/internal/services"
	"cityio/internal/stream"
)

type userActor struct {
	baseActor
	User domain.User

	// foodIncomeAccum and foodUpkeepAccum sum food flowing in/out of the user
	// pool between samples. On each periodic tick they are converted to
	// per-second rates and zeroed.
	foodIncomeAccum int64
	foodUpkeepAccum int64

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
		slog.DebugContext(state.Ctx(), "registering user actor", "username", msg.User.Username)
		state.User = msg.User
		if !msg.Restore {
			if err := state.Store.CreateUser(state.Ctx(), state.User); err != nil {
				slog.ErrorContext(state.Ctx(), "failed to persist user create", "user_id", state.User.UserID, "error", err)
			}
			services.CreateCity(state.Ctx(), state.Cluster, state.Store, &services.CityInput{ //nolint:errcheck // fire-and-forget
				Type:  domain.CityTypeCity,
				Owner: &state.User.UserID,
				Name:  fmt.Sprintf("%s's City", state.User.Username),
				Size:  constants.CitySize,
			})
		}
		state.startPeriodicOperation(ctx)
		ctx.Respond(messages.Ack{})

	case messages.CreditUserMessage:
		state.User.Gold += msg.Gold
		state.User.Food += msg.Food
		state.ws()
		ctx.Respond(messages.Ack{})

	case messages.DepositFoodMessage:
		if msg.Amount > 0 {
			state.User.Food += msg.Amount
			state.foodIncomeAccum += msg.Amount
			state.ws()
		}

	case messages.RequestFoodFromPoolMessage:
		granted := max(min(msg.Amount, state.User.Food), 0)
		state.User.Food -= granted
		state.foodUpkeepAccum += granted
		if granted > 0 {
			state.ws()
		}
		// TODO: when players can own multiple cities, batch requests within a
		// window and allocate by priority (capital first, then by population
		// descending) instead of first-come.
		ctx.Respond(messages.RequestFoodFromPoolResponse{Granted: granted})

	case messages.CheckAndDeductGoldMessage:
		if missing := msg.Amount - state.User.Gold; missing > 0 {
			ctx.Respond(messages.InsufficientGoldError{
				Missing: missing,
			})
			return
		}
		state.User.Gold -= msg.Amount
		state.ws()
		ctx.Respond(messages.Ack{})

	case messages.GetUserMessage:
		ctx.Respond(&messages.GetUserResponseMessage{
			User: state.User,
		})

	case messages.DeleteUserMessage:
		if err := state.Store.DeleteUser(state.Ctx(), state.User.UserID); err != nil {
			slog.ErrorContext(state.Ctx(), "failed to delete user", "user_id", state.User.UserID, "error", err)
		}

		slog.DebugContext(state.Ctx(), "shutting down user actor", "user_id", state.User.UserID)
		state.stopPeriodicOperation()
		ctx.Stop(ctx.Self())

	case messages.PeriodicOperationMessage:
		seconds := float64(constants.UserBackupFrequency)
		state.User.FoodIncomeRate = float64(state.foodIncomeAccum) / seconds
		state.User.FoodUpkeepRate = float64(state.foodUpkeepAccum) / seconds
		state.foodIncomeAccum = 0
		state.foodUpkeepAccum = 0
		state.Store.EnqueueUser(state.User)
		state.ws()
	}
}

func (state *userActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.UserBackupFrequency * time.Second)
	state.stopTickerCh = make(chan struct{})

	pid := ctx.Self()
	system := ctx.ActorSystem()
	go func() {
		for {
			select {
			case <-state.ticker.C:
				system.Root.Send(pid, messages.PeriodicOperationMessage{})
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
	u := state.User
	stream.Publish(state.User.UserID, stream.StateUpdate{User: &u})
}
