package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
)

type cityCenterImpl struct{}

func newCityCenterImpl() buildingActorImpl {
	return &cityCenterImpl{}
}

func (c *cityCenterImpl) Create(ctx actor.Context, state *buildingActor)  {}
func (c *cityCenterImpl) Destroy(ctx actor.Context, state *buildingActor) {}
func (c *cityCenterImpl) Handle(ctx actor.Context, state *buildingActor) {
	switch ctx.Message().(type) {

	case messages.PeriodicOperationMessage:
		if state.constructionActive() || state.Owner == nil {
			return
		}

		err := state.Cluster.Tell("user", *state.Owner, messages.UpdateUserGoldMessage{
			Change: constants.GetBuildingProduction(state.Building.BuildingType(), state.Building.Level),
		})
		if err != nil {
			state.Log.Error("failed to send city center production back to user", "error", err)
		}
	}
}
