package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
)

type mineImpl struct{}

func newMineImpl() buildingActorImpl {
	return &mineImpl{}
}

func (c *mineImpl) Create(ctx actor.Context, state *buildingActor)  {}
func (c *mineImpl) Destroy(ctx actor.Context, state *buildingActor) {}
func (c *mineImpl) Handle(ctx actor.Context, state *buildingActor) {
	switch ctx.Message().(type) {

	case messages.PeriodicOperationMessage:
		if state.constructionActive() {
			return
		}
		perDay := constants.GetBuildingProduction(state.Building.BuildingType(), state.Building.Level, "gold")
		state.creditProduction(constants.PerTickAmount(perDay, constants.BuildingProductionFrequency), 0)
	}
}
