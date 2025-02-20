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
			ctx.Send(state.database, messages.CreateBuildingMessage{
				Building: state.Building,
			})

			response, err := Request[messages.UpdateCityPopulationCapResponseMessage](ctx, state.getCityPID(), messages.UpdateCityPopulationCapMessage{
				Change: constants.GetBuildingPopulation(constants.BUILDING_TYPE_CITY_CENTER, state.Building.Level),
			})
			if err != nil {
				log.Printf("Error updating city population cap: %s", err)
				ctx.Respond(messages.CreateBuildingResponseMessage{
					Error: err,
				})
				return
			}
			if response.Error != nil {
				log.Printf("Error updating city population cap: %s", response.Error)
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

	case messages.UpgradeBuildingMessage:
		ctx.Respond(messages.UpgradeBuildingResponseMessage{
			Error: state.upgradeBuilding(ctx),
		})

	case messages.PeriodicOperationMessage:
		if state.Building.ConstructionEnd.After(time.Now()) {
			return
		}

		userPID := state.getUserPID()
		if userPID == nil {
			// not owned by a player, don't update production balance
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

	case messages.GetBuildingMessage:
		ctx.Respond(messages.GetBuildingResponseMessage{
			Building: state.Building,
		})

	case messages.DeleteBuildingMessage:
		state.stopPeriodicOperation()
		state.deleteBuilding(ctx)
	}
}

func (state *CityCenterActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.BUILDING_PRODUCTION_FREQUENCY * time.Second)
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

func (state *CityCenterActor) stopPeriodicOperation() {
	close(state.stopTickerCh)
	state.ticker = nil
}
