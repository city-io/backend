package actors

import (
	"cityio/internal/constants"
	"cityio/internal/messages"

	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

type CityCenterActor struct {
	BuildingActor

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func (state *CityCenterActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateBuildingMessage:
		state.Building = msg.Building

		if !msg.Restore {
			err := state.createCityCenter()
			if err != nil {
				ctx.Respond(messages.CreateBuildingResponseMessage{
					Error: err,
				})
				return
			}

			response, err := Request[messages.UpdateCityPopulationResponseMessage](ctx, state.getUserPID(), messages.UpdateCityPopulationMessage{
				// TODO: Update to separate building population modifier
				Change: constants.GetBuildingProduction(constants.BUILDING_TYPE_CITY_CENTER, state.Building.Level),
			})
			if err != nil {
				log.Printf("Error updating city population: %s", err)
				ctx.Respond(messages.CreateBuildingResponseMessage{
					Error: err,
				})
				return
			}
			if response.Error != nil {
				log.Printf("Error updating city population: %s", response.Error)
				ctx.Respond(messages.CreateBuildingResponseMessage{
					Error: response.Error,
				})
				return
			}
		}
		ctx.Respond(messages.CreateBuildingResponseMessage{
			Error: nil,
		})
		state.startPeriodicOperation(ctx)

	case messages.PeriodicOperationMessage:
		userPID := state.getUserPID()

		// handle population growth event here

		if userPID == nil {
			// not owned by a player
			return
		}
		updateGoldResponse, err := Request[messages.UpdateUserGoldResponseMessage](ctx, userPID, messages.UpdateUserGoldMessage{
			Change: constants.GetBuildingProduction(constants.BUILDING_TYPE_CITY_CENTER, state.Building.Level),
		})
		if err != nil {
			log.Printf("Error updating user gold: %s", err)
		}
		if updateGoldResponse.Error != nil {
			log.Printf("Error updating user gold: %s", updateGoldResponse.Error)
		}

		var updateFoodResponse *messages.UpdateUserFoodResponseMessage
		updateFoodResponse, err = Request[messages.UpdateUserFoodResponseMessage](ctx, userPID, messages.UpdateUserFoodMessage{
			Change: constants.GetBuildingProduction(constants.BUILDING_TYPE_CITY_CENTER, state.Building.Level),
		})
		if err != nil {
			log.Printf("Error updating user gold: %s", err)
		}
		if updateFoodResponse.Error != nil {
			log.Printf("Error updating user gold: %s", updateFoodResponse.Error)
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

func (state *CityCenterActor) createCityCenter() error {
	result := state.db.Create(&state.Building)
	if result.Error != nil {
		log.Printf("Error creating city center: %s", result.Error)
		return result.Error
	}
	return nil
}

func (state *CityCenterActor) startPeriodicOperation(ctx actor.Context) {
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

func (state *CityCenterActor) stopPeriodicOperation() {
	close(state.stopTickerCh)
	state.ticker = nil
}
