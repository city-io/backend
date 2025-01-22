package actors

import (
	"cityio/internal/constants"
	"cityio/internal/messages"

	"log"

	"github.com/asynkron/protoactor-go/actor"
)

type HouseActor struct {
	BuildingActor
}

func (state *HouseActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateBuildingMessage:
		state.Building = msg.Building
		if !msg.Restore {
			ctx.Send(state.database, messages.CreateBuildingMessage{
				Building: state.Building,
			})

			response, err := Request[messages.UpdateCityPopulationCapResponseMessage](ctx, state.getUserPID(), messages.UpdateCityPopulationCapMessage{
				Change: constants.GetBuildingPopulation(constants.BUILDING_TYPE_HOUSE, state.Building.Level),
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

	case messages.UpgradeBuildingMessage:
		ctx.Respond(messages.UpgradeBuildingResponseMessage{
			Error: state.upgradeBuilding(ctx),
		})

	case messages.GetBuildingMessage:
		ctx.Respond(messages.GetBuildingResponseMessage{
			Building: state.Building,
		})

	case messages.DeleteBuildingMessage:
		state.deleteBuilding(ctx)
	}
}
