package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

type BuildingActor struct {
	BaseActor
	Building models.Building
	OwnerId  string

	CityPID    *actor.PID
	MapTilePID *actor.PID
	UserPID    *actor.PID

	once sync.Once
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

func (state *BuildingActor) getCityPID() *actor.PID {
	state.once.Do(func() {
		response, err := Request[messages.GetCityPIDResponseMessage](system.Root, GetManagerPID(), messages.GetCityPIDMessage{
			CityId: state.Building.CityId,
		})
		if err != nil {
			log.Printf("Error getting city pid: %s", err)
			return
		}
		if response.PID == nil {
			log.Printf("City pid is nil")
		} else {
			state.CityPID = response.PID
		}
	})

	return state.CityPID
}

func (state *BuildingActor) getUserPID() *actor.PID {
	cityPID := state.getCityPID()
	if cityPID == nil {
		return nil
	}

	getCityResponse, err := Request[messages.GetCityResponseMessage](system.Root, cityPID, messages.GetCityMessage{})
	if err != nil {
		log.Printf("Error getting city: %s", err)
		return nil
	}
	if getCityResponse.City.Owner != state.OwnerId {
		var getUserPIDResponse *messages.GetUserPIDResponseMessage
		getUserPIDResponse, err = Request[messages.GetUserPIDResponseMessage](system.Root, GetManagerPID(), messages.GetUserPIDMessage{
			UserId: getCityResponse.City.Owner,
		})
		if err != nil {
			log.Printf("Error getting user pid: %s", err)
			return nil
		}
		state.UserPID = getUserPIDResponse.PID
	}
	return state.UserPID
}
