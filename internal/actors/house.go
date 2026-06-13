package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/messages"
)

type houseImpl struct{}

func newHouseImpl() buildingActorImpl {
	return &houseImpl{}
}

func (c *houseImpl) Create(ctx actor.Context, state *buildingActor) {
	state.reportPopulation(constants.GetBuildingPopulation(domain.BuildingTypeHouse, state.populationLevel()))
}

func (c *houseImpl) Destroy(ctx actor.Context, state *buildingActor) {}

func (c *houseImpl) Handle(ctx actor.Context, state *buildingActor) {
	switch ctx.Message().(type) {

	case messages.PeriodicOperationMessage:
		if state.constructionActive() {
			return
		}
		state.reportPopulation(constants.GetBuildingPopulation(domain.BuildingTypeHouse, state.populationLevel()))
	}
}
