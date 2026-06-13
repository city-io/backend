package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
)

type barracksImpl struct{}

func newBarracksImpl() buildingActorImpl {
	return &barracksImpl{}
}

func (b *barracksImpl) Create(ctx actor.Context, state *buildingActor)  {}
func (b *barracksImpl) Destroy(ctx actor.Context, state *buildingActor) {}
func (b *barracksImpl) Handle(ctx actor.Context, state *buildingActor) {
	switch ctx.Message().(type) {

	case messages.PeriodicOperationMessage:
		if state.constructionActive() {
			return
		}
		state.creditProduction(constants.GetBuildingProduction(state.Building.BuildingType(), state.Building.Level), 0)
	}
}
