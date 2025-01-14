package actors

import (
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
		state.CityPID = msg.CityPID
		state.UserPID = msg.UserPID

		if !msg.Restore {
			err := state.createMine()
			ctx.Respond(messages.CreateBuildingResponseMessage{
				Error: err,
			})
		} else {
			ctx.Respond(messages.CreateBuildingResponseMessage{
				Error: nil,
			})
		}
		state.startPeriodicOperation(ctx)

	case messages.PeriodicOperationMessage:
		// TODO: set constant amount based on level
		response, err := Request[messages.UpdateUserGoldResponseMessage](ctx, state.getUserPID(), messages.UpdateUserGoldMessage{
			Change: 1,
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

func (state *MineActor) createMine() error {
	result := state.db.Create(&state.Building)
	if result.Error != nil {
		log.Printf("Error creating mine: %s", result.Error)
		return result.Error
	}
	return nil
}

func (state *MineActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(3 * time.Second)

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
