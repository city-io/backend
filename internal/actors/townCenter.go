package actors

import (
	"log/slog"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
)

type townCenterImpl struct{}

func newTownCenterImpl() buildingActorImpl {
	return &townCenterImpl{}
}

func (c *townCenterImpl) Create(ctx actor.Context, state *buildingActor)  {}
func (c *townCenterImpl) Destroy(ctx actor.Context, state *buildingActor) {}
func (c *townCenterImpl) Handle(ctx actor.Context, state *buildingActor) {
	switch ctx.Message().(type) {

	case messages.PeriodicOperationMessage:
		if state.constructionActive() || state.Owner == nil {
			return
		}

		err := state.Cluster.Tell("user", *state.Owner, messages.UpdateUserGoldMessage{
			Change: constants.GetBuildingProduction(state.Building.BuildingType(), state.Building.Level),
		})
		if err != nil {
			slog.ErrorContext(state.Ctx(), "failed to send town center production back to user", "error", err)
		}
	}
}
