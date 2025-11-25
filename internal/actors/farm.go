package actors

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/ports"
)

type farmActor struct {
	BuildingActor

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func NewFarmActor() ports.BaseActorInterface {
	return &farmActor{}
}

func (state *farmActor) ActorType() string {
	return string(constants.BuildingTypeFarm)
}

func (state *farmActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateBuildingMessage:
		state.Building = msg.Building
		if !msg.Restore {
			ctx.Send(state.Cluster.DB(), messages.CreateBuildingMessage{
				Building: state.Building,
			})
		}
		state.startPeriodicOperation(ctx)
		ctx.Respond(messages.Ack{})

	case messages.UpgradeBuildingMessage:
		state.upgrade(ctx)

	case messages.PeriodicOperationMessage:
		if state.constructionActive() || state.Owner == nil {
			return
		}

		err := state.Cluster.Tell("user", *state.Owner, messages.UpdateUserFoodMessage{
			Change: constants.GetBuildingProduction(state.Building.BuildingType(), state.Building.Level),
		})
		if err != nil {
			state.Log.Error("failed to send farm production back to user", "error", err)
		}

	case messages.GetBuildingMessage:
		ctx.Respond(messages.GetBuildingResponseMessage{
			Building: state.Building,
		})

	case messages.DeleteBuildingMessage:
		state.stopPeriodicOperation()
		state.destroy(ctx)
	}
}

func (state *farmActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.BuildingProductionFrequency * time.Second)
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

func (state *farmActor) stopPeriodicOperation() {
	select {
	case <-state.stopTickerCh:
	default:
		close(state.stopTickerCh)
	}
}
