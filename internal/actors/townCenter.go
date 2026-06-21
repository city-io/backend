package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/messages"
)

type townCenterImpl struct{}

func newTownCenterImpl() buildingActorImpl {
	return &townCenterImpl{}
}

func (c *townCenterImpl) Create(ctx actor.Context, state *buildingActor) {
	state.reportPopulation(constants.GetBuildingPopulation(domain.BuildingTypeTownCenter, state.populationLevel()))
}
func (c *townCenterImpl) Destroy(ctx actor.Context, state *buildingActor) {}
func (c *townCenterImpl) Handle(ctx actor.Context, state *buildingActor) {
	switch ctx.Message().(type) {

	case messages.PeriodicOperationMessage:
		if state.constructionActive() {
			return
		}
		state.reportPopulation(constants.GetBuildingPopulation(domain.BuildingTypeTownCenter, state.populationLevel()))
		perDay := constants.GetBuildingProduction(state.Building.BuildingType(), state.Building.Level, "gold")
		state.creditProduction(constants.PerTickAmount(perDay, constants.BuildingTickInterval), 0)
	}
}
