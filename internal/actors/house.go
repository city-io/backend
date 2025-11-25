package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
)

type houseImpl struct{}

func newHouseImpl() buildingActorImpl {
	return &houseImpl{}
}

func (c *houseImpl) Create(ctx actor.Context, state *buildingActor) {
	// TODO: switch this to an on-upgrade/construction complete hook
	err := state.Cluster.Tell("city", state.Building.CityID, messages.UpdateCityPopulationCapMessage{
		Change: constants.GetBuildingPopulation(constants.BuildingTypeHouse, 1),
	})
	if err != nil {
		state.Log.Error("failed to increment city population cap from house construction", "error", err)
	}
}

func (c *houseImpl) Destroy(ctx actor.Context, state *buildingActor) {}

func (c *houseImpl) Handle(ctx actor.Context, state *buildingActor) {
	switch ctx.Message().(type) {

	case messages.PeriodicOperationMessage:
		if state.constructionActive() || state.Owner == nil {
			return
		}
	}
}
