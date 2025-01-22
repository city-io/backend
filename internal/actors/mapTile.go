package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

type army struct {
	ArmyPID *actor.PID
	Army    models.Army
}

type MapTileActor struct {
	BaseActor
	Tile models.MapTile

	CityPID     *actor.PID
	BuildingPID *actor.PID
	Armies      map[string][]*army

	cityOnce sync.Once
}

func (state *MapTileActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateMapTileMessage:
		state.Tile = msg.Tile
		state.Armies = make(map[string][]*army)
		if !msg.Restore {
			ctx.Send(state.database, messages.CreateMapTileMessage{
				Tile: state.Tile,
			})
		}
		ctx.Respond(messages.CreateMapTileResponseMessage{
			Error: nil,
		})

	case messages.AddTileArmyMessage:
		// no armies from player on this tile
		if _, ok := state.Armies[msg.Army.Owner]; !ok {
			state.Armies[msg.Army.Owner] = append(make([]*army, 0), &army{
				ArmyPID: msg.ArmyPID,
				Army:    msg.Army,
			})
		} else {
			log.Printf("Merging idle armies at %d, %d", state.Tile.X, state.Tile.Y)
			state.Armies[msg.Army.Owner] = append(state.Armies[msg.Army.Owner], &army{
				ArmyPID: msg.ArmyPID,
				Army:    msg.Army,
			})
			mergeArmies := make([]*army, 0)
			newArmies := make([]*army, 0)
			for i := 0; i < len(state.Armies[msg.Army.Owner]); i++ {
				if !state.Armies[msg.Army.Owner][i].Army.MarchActive {
					mergeArmies = append(mergeArmies, state.Armies[msg.Army.Owner][i])
				} else {
					newArmies = append(newArmies, state.Armies[msg.Army.Owner][i])
				}
			}
			// merge together all armies that are not marching anywhere
			if len(mergeArmies) > 1 {
				mergedArmy := &army{
					ArmyPID: mergeArmies[0].ArmyPID,
					Army:    mergeArmies[0].Army,
				}
				for i := 1; i < len(mergeArmies); i++ {
					mergedArmy.Army.Size += mergeArmies[i].Army.Size
					deleteArmyResponse, err := Request[messages.DeleteArmyResponseMessage](ctx, mergeArmies[i].ArmyPID, messages.DeleteArmyMessage{
						ArmyId: mergeArmies[i].Army.ArmyId,
					})
					if err != nil {
						log.Printf("Error merging army: %s", err)
						return
					}
					if deleteArmyResponse.Error != nil {
						log.Printf("Error merging army: %s", deleteArmyResponse.Error)
						return
					}
				}
				updateArmyMessage, err := Request[messages.UpdateArmyResponseMessage](ctx, mergedArmy.ArmyPID, messages.UpdateArmyMessage{
					Army: mergedArmy.Army,
				})
				if err != nil {
					log.Printf("Error merging army: %s", err)
					return
				}
				if updateArmyMessage.Error != nil {
					log.Printf("Error merging army: %s", updateArmyMessage.Error)
					return
				}
				newArmies = append([]*army{mergedArmy}, newArmies...)
				state.Armies[msg.Army.Owner] = newArmies
			}
		}

	case messages.RemoveTileArmyMessage:
		if len(state.Armies[msg.Owner]) == 1 {
			delete(state.Armies, msg.Owner)
		} else {
			newArmies := make([]*army, 0)
			for _, army := range state.Armies[msg.Owner] {
				if army.Army.ArmyId != msg.ArmyId {
					newArmies = append(newArmies, army)
				}
			}
			state.Armies[msg.Owner] = newArmies
		}

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
		armies := make(map[string][]*models.Army)
		for owner, _armies := range state.Armies {
			newArmies := make([]*models.Army, 0)
			for _, army := range _armies {
				newArmies = append(newArmies, &army.Army)
			}
			armies[owner] = newArmies
		}
		ctx.Respond(messages.GetMapTileArmiesResponseMessage{
			Armies: armies,
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
