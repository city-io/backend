package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/messages"
)

type cityCenterImpl struct{}

func newCityCenterImpl() buildingActorImpl {
	return &cityCenterImpl{}
}

func (c *cityCenterImpl) Create(ctx actor.Context, state *buildingActor) {
	state.reportPopulation(constants.GetBuildingPopulation(domain.BuildingTypeCityCenter, state.populationLevel()))
}
func (c *cityCenterImpl) Destroy(ctx actor.Context, state *buildingActor) {}
func (c *cityCenterImpl) Handle(ctx actor.Context, state *buildingActor) {
	switch ctx.Message().(type) {

	case messages.PeriodicOperationMessage:
		state.reportPopulation(constants.GetBuildingPopulation(domain.BuildingTypeCityCenter, state.populationLevel()))
		state.creditProduction(state.productionForTick("gold"), 0)
	}
}
