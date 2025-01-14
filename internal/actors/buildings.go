package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
)

type BuildingActor struct {
	BaseActor
	Building   models.Building
	CityPID    *actor.PID
	MapTilePID *actor.PID
	UserPID    *actor.PID
}

func (state *BuildingActor) getBuilding(ctx actor.Context) {
	ctx.Respond(messages.GetBuildingResponseMessage{
		Building: state.Building,
	})
}

func (state *BuildingActor) deleteBuilding(ctx actor.Context) {
	result := state.db.Delete(&state.Building)
	if result.Error != nil {
		log.Printf("Error deleting building: %s", result.Error)
	}
	ctx.Respond(messages.DeleteBuildingResponseMessage{
		Error: result.Error,
	})
	log.Printf("Shutting down BarracksActor at: (%d, %d)", state.Building.X, state.Building.Y)
	ctx.Stop(ctx.Self())
}

// add helpers to update PIDs when ids change
