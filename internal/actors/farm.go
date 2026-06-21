package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
)

type farmImpl struct{}

func newFarmImpl() buildingActorImpl {
	return &farmImpl{}
}

func (c *farmImpl) Create(ctx actor.Context, state *buildingActor)  {}
func (c *farmImpl) Destroy(ctx actor.Context, state *buildingActor) {}
func (c *farmImpl) Handle(ctx actor.Context, state *buildingActor) {
	switch ctx.Message().(type) {

	case messages.PeriodicOperationMessage:
		if state.constructionActive() {
			return
		}
		perDay := constants.GetBuildingProduction(state.Building.BuildingType(), state.Building.Level, "food")
		state.creditProduction(0, constants.PerTickAmount(perDay, constants.BuildingTickInterval))
	}
}
