package actors

import (
	"log/slog"

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
		if state.constructionActive() || state.Owner == nil {
			return
		}

		err := state.Cluster.Tell("user", *state.Owner, messages.UpdateUserFoodMessage{
			Change: constants.GetBuildingProduction(state.Building.BuildingType(), state.Building.Level),
		})
		if err != nil {
			slog.ErrorContext(state.Ctx(), "failed to send farm production back to user", "error", err)
		}
	}
}
