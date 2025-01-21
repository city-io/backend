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

func (state *BuildingActor) createBuilding(ctx actor.Context) {
	ctx.Send(state.database, messages.CreateBuildingMessage{
		Building: state.Building,
	})
}

func (state *BuildingActor) deleteBuilding(ctx actor.Context) {
	ctx.Send(state.database, messages.DeleteBuildingMessage{
		BuildingId: state.Building.BuildingId,
	})
	ctx.Respond(messages.DeleteBuildingResponseMessage{
		Error: nil,
	})
	log.Printf("Shutting down BuildingActor of type %s at: (%d, %d)", state.Building.Type, state.Building.X, state.Building.Y)
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
			log.Printf("City PID is nil")
		} else {
			state.CityPID = response.PID
		}
	})

	return state.CityPID
}

func (state *BuildingActor) getOwnerId() (string, error) {
	cityPID := state.getCityPID()
	if cityPID == nil {
		log.Println("City PID is nil")
		return "", &messages.CityNotFoundError{CityId: state.Building.CityId}
	}

	getCityResponse, err := Request[messages.GetCityResponseMessage](system.Root, cityPID, messages.GetCityMessage{})
	if err != nil {
		log.Printf("Error getting city: %s", err)
		return "", err
	}
	if getCityResponse.City.Owner == "" {
		return "", nil
	}
	return getCityResponse.City.Owner, nil
}

func (state *BuildingActor) getUserPID() *actor.PID {
	ownerId, err := state.getOwnerId()
	if err != nil {
		return nil
	}
	if ownerId != state.OwnerId {
		var getUserPIDResponse *messages.GetUserPIDResponseMessage
		getUserPIDResponse, err = Request[messages.GetUserPIDResponseMessage](system.Root, GetManagerPID(), messages.GetUserPIDMessage{
			UserId: ownerId,
		})
		if err != nil {
			log.Printf("Error getting user pid: %s", err)
			return nil
		}
		state.UserPID = getUserPIDResponse.PID
		state.OwnerId = ownerId
	}
	return state.UserPID
}
