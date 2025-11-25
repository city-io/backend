package actors

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/ports"
)

type houseActor struct {
	BuildingActor

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func NewHouseActor() ports.BaseActorInterface {
	return &houseActor{}
}

func (state *houseActor) ActorType() string {
	return string(constants.BuildingTypeHouse)
}

func (state *houseActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateBuildingMessage:
		state.Building = msg.Building
		if !msg.Restore {
			ctx.Send(state.Cluster.DB(), messages.CreateBuildingMessage{
				Building: state.Building,
			})

			// TODO: move this to happen on construction complete
			err := state.Cluster.Tell("city", state.Building.CityID, messages.UpdateCityPopulationCapMessage{
				Change: constants.GetBuildingPopulation(constants.BuildingTypeHouse, 1),
			})
			if err != nil {
				state.Log.Error("failed to increment city population cap from house construction", "error", err)
			}
		}
		state.startPeriodicOperation(ctx)
		ctx.Respond(messages.Ack{})

	case messages.UpgradeBuildingMessage:
		state.upgrade(ctx)

	case messages.PeriodicOperationMessage:
		if state.constructionActive() || state.Owner == nil {
			return
		}
		// TODO: do houses have any periodic activity?

	case messages.GetBuildingMessage:
		ctx.Respond(messages.GetBuildingResponseMessage{
			Building: state.Building,
		})

	case messages.DeleteBuildingMessage:
		state.stopPeriodicOperation()
		state.destroy(ctx)
	}
}

func (state *houseActor) startPeriodicOperation(ctx actor.Context) {
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

func (state *houseActor) stopPeriodicOperation() {
	select {
	case <-state.stopTickerCh:
	default:
		close(state.stopTickerCh)
	}
}
