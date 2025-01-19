package actors

import (
	"cityio/internal/constants"
	"cityio/internal/messages"

	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

type MineActor struct {
	BuildingActor

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func (state *MineActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateBuildingMessage:
		state.Building = msg.Building
		if !msg.Restore {
			ctx.Send(state.database, messages.CreateBuildingMessage{
				Building: state.Building,
			})
		}
		ctx.Respond(messages.CreateBuildingResponseMessage{
			Error: nil,
		})
		state.startPeriodicOperation(ctx)

	case messages.PeriodicOperationMessage:
		userPID := state.getUserPID()
		if userPID == nil {
			// not owned by a player
			return
		}
		response, err := Request[messages.UpdateUserGoldResponseMessage](ctx, userPID, messages.UpdateUserGoldMessage{
			Change: constants.GetBuildingProduction(state.Building.Type, state.Building.Level),
		})
		if err != nil {
			log.Printf("Error updating user gold: %s", err)
		}
		if response.Error != nil {
			log.Printf("Error updating user gold: %s", response.Error)
		}

	case messages.UpdateBuildingTilePIDMessage:
		state.MapTilePID = msg.TilePID

	case messages.GetBuildingMessage:
		state.getBuilding(ctx)

	case messages.DeleteBuildingMessage:
		state.stopPeriodicOperation()
		state.deleteBuilding(ctx)
	}
}

func (state *MineActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.BUILDING_PRODUCTION_FREQUENCY * time.Second)

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

func (state *MineActor) stopPeriodicOperation() {
	close(state.stopTickerCh)
	state.ticker = nil
}
