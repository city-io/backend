package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

type MapTileActor struct {
	BaseActor
	Tile models.MapTile

	CityPID     *actor.PID
	BuildingPID *actor.PID
	ArmyPIDs    []*actor.PID
	Armies      []models.Army

	cityOnce sync.Once
}

func (state *MapTileActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateMapTileMessage:
		state.Tile = msg.Tile
		state.ArmyPIDs = make([]*actor.PID, 0)
		if !msg.Restore {
			ctx.Send(state.database, messages.CreateMapTileMessage{
				Tile: state.Tile,
			})
		}
		ctx.Respond(messages.CreateMapTileResponseMessage{
			Error: nil,
		})

	case messages.AddTileArmyMessage:
		state.ArmyPIDs = append(state.ArmyPIDs, msg.ArmyPID)
		state.Armies = append(state.Armies, msg.Army)
		ctx.Respond(messages.AddTileArmyResponseMessage{
			Error: nil,
		})

	case messages.GetMapTileMessage:
		var city *models.City = nil
		if state.CityPID != nil {
			response, err := Request[messages.GetCityResponseMessage](ctx, state.CityPID, messages.GetCityMessage{})
			if err != nil {
				log.Printf("Error getting city: %s", err)
			} else {
				city = &response.City
			}
		}
		var building *models.Building = nil
		if state.BuildingPID != nil {
			response, err := Request[messages.GetBuildingResponseMessage](ctx, state.BuildingPID, messages.GetBuildingMessage{})
			if err != nil {
				log.Printf("Error getting building: %s", err)
			} else {
				building = &response.Building
			}
		}
		ctx.Respond(messages.GetMapTileResponseMessage{
			Tile:     state.Tile,
			City:     city,
			Building: building,
		})

	case messages.GetMapTileArmiesMessage:
		ctx.Respond(messages.GetMapTileArmiesResponseMessage{
			Armies: state.Armies,
		})
	}
}

func (state *MapTileActor) getCityPID() (*actor.PID, error) {
	// can cities ever be removed? if so this should not use sync.Once
	state.cityOnce.Do(func() {
		getCityPIDResponse, err := Request[messages.GetCityPIDResponseMessage](system.Root, GetManagerPID(), messages.GetCityPIDMessage{
			CityId: state.Tile.CityId,
		})
		if err != nil {
			log.Printf("Error restoring map tile: %s", err)
			return
		}
		if getCityPIDResponse.PID == nil {
			log.Printf("Error restoring map tile: City not found")
			return
		}
		state.CityPID = getCityPIDResponse.PID
	})
	if state.CityPID == nil {
		return nil, &messages.CityNotFoundError{CityId: state.Tile.CityId}
	}
	return state.CityPID, nil
}

func (state *MapTileActor) getBuildingPID() (*actor.PID, error) {
	getBuildingPIDResponse, err := Request[messages.GetBuildingPIDResponseMessage](system.Root, GetManagerPID(), messages.GetBuildingPIDMessage{
		BuildingId: state.Tile.BuildingId,
	})
	if err != nil {
		log.Printf("Error retrieving map tile building pid: %s", err)
		return nil, err
	}
	if getBuildingPIDResponse.PID == nil {
		log.Printf("Error retrieving map tile building pid: Building not found")
		return nil, nil
	}
	return getBuildingPIDResponse.PID, nil
}
