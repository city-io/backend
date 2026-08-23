package actors

import (
	"github.com/asynkron/protoactor-go/actor"

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
		state.creditProduction(state.productionForTick("gold"), 0)
	}
}
