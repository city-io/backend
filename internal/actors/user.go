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
				ctx.Respond(&messages.UserCreationError{UserID: state.User.UserID})
				return
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
		// Background production credit (gold from mines/centers, etc).
		// State changes but we don't publish here — many buildings hit this
		// per tick, and clients only care about the net amount over a window.
		// The periodic tick below publishes the consolidated state.
		state.User.Gold += msg.Gold
		state.User.Food += msg.Food
		ctx.Respond(messages.Ack{})

	case messages.DepositFoodMessage:
		// City surplus flowing into the pool. Same batching reasoning as
		// CreditUserMessage — the periodic publish carries the net change.
		if msg.Amount > 0 {
			state.User.Food += msg.Amount
			state.foodIncomeAccum += msg.Amount
		}

	case messages.RequestFoodFromPoolMessage:
		// City deficit drawing from the pool. Same batching reasoning.
		granted := max(min(msg.Amount, state.User.Food), 0)
		state.User.Food -= granted
		state.foodUpkeepAccum += granted
		// TODO: when players can own multiple cities, batch requests within a
		// window and allocate by priority (capital first, then by population
		// descending) instead of first-come.
		ctx.Respond(messages.RequestFoodFromPoolResponse{Granted: granted})

	case messages.CheckAndDeductGoldMessage:
		// Player-initiated spend (e.g. building upgrade). Publish immediately
		// so the player sees the deduction in their HUD without waiting for
		// the next periodic tick. The publish also carries any accumulated
		// background credits, so they appear at the same moment too — feels
		// natural rather than "my gold just jumped between actions."
		if missing := msg.Amount - state.User.Gold; missing > 0 {
			ctx.Respond(messages.InsufficientGoldError{
				Missing: missing,
			})
			return
		}
		state.User.Gold -= msg.Amount
		state.publish()
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
		windowSecs := int64(constants.UserBackupFrequency)
		state.User.FoodIncomeRate = state.foodIncomeAccum * int64(constants.SecondsPerHour) / windowSecs
		state.User.FoodUpkeepRate = state.foodUpkeepAccum * int64(constants.SecondsPerHour) / windowSecs
		state.foodIncomeAccum = 0
		state.foodUpkeepAccum = 0
		state.Store.EnqueueUser(state.User)
		state.publish()
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

// publish pushes the user's current state to their StreamState
// subscribers via the in-process pub/sub. Call after any change the player
// should see without waiting for the next periodic tick — gold/food balance
// shifts, deposit/withdrawal, etc.
func (state *userActor) publish() {
	u := state.User
	stream.Publish(state.User.UserID, stream.StateUpdate{User: &u})
}
