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

	case messages.AddCityToTileMessage:
		state.Tile.CityId = msg.CityId
		ctx.Respond(messages.AddCityToTileResponseMessage{
			Error: nil,
		})

	case messages.AddBuildingToTileMessage:
		state.Tile.BuildingId = msg.BuildingId
		ctx.Respond(messages.AddBuildingToTileResponseMessage{
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
		cityPID, err := state.getCityPID()
		if err != nil {
			log.Printf("Error getting city pid: %s", err)
			ctx.Respond(messages.GetMapTileResponseMessage{
				Tile:     state.Tile,
				City:     nil,
				Building: nil,
			})
			return
		}
		if cityPID != nil {
			response, err := Request[messages.GetCityResponseMessage](ctx, cityPID, messages.GetCityMessage{})
			if err != nil {
				log.Printf("Error getting city: %s", err)
			} else {
				city = &response.City
			}
		}

		var building *models.Building = nil
		buildingPID, err := state.getBuildingPID()
		if err != nil {
			log.Printf("Error getting building pid: %s", err)
			ctx.Respond(messages.GetMapTileResponseMessage{
				Tile:     state.Tile,
				City:     city,
				Building: nil,
			})
			return
		}
		if buildingPID != nil {
			response, err := Request[messages.GetBuildingResponseMessage](ctx, buildingPID, messages.GetBuildingMessage{})
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
			Armies:   state.getTileArmies(),
		})

	case messages.GetMapTileArmiesMessage:
		// returns names of owners and their armies
		armies := state.getTileArmies()
		ctx.Respond(messages.GetMapTileArmiesResponseMessage{
			Armies: armies,
		})
		// below code only returns userIds
		// armies := make(map[string][]*models.Army)
		// for owner, _armies := range state.Armies {
		// 	newArmies := make([]*models.Army, 0)
		// 	for _, army := range _armies {
		// 		newArmies = append(newArmies, &army.Army)
		// 	}
		// 	armies[owner] = newArmies
		// }
		// ctx.Respond(messages.GetMapTileArmiesResponseMessage{
		// 	Armies: armies,
		// })
	}
}

func (state *MapTileActor) getTileArmies() map[string][]*models.Army {
	// TODO: add better error handling
	ownerNames := make(map[string]string)
	armies := make(map[string][]*models.Army)
	for owner, _armies := range state.Armies {
		newArmies := make([]*models.Army, 0)
		for _, army := range _armies {
			newArmies = append(newArmies, &army.Army)
		}
		if _, ok := ownerNames[owner]; !ok {
			getUserPIDResponse, err := Request[messages.GetUserPIDResponseMessage](system.Root, GetManagerPID(), messages.GetUserPIDMessage{
				UserId: owner,
			})
			if err != nil {
				log.Printf("Error getting user pid: %s", err)
				return armies
			}
			if getUserPIDResponse.PID == nil {
				log.Printf("User pid not found for %s", owner)
				return armies
			}
			getUserResponse, err := Request[messages.GetUserResponseMessage](system.Root, getUserPIDResponse.PID, messages.GetUserMessage{})
			if err != nil {
				log.Printf("Error getting user: %s", err)
				return armies
			}
			ownerNames[owner] = getUserResponse.User.Username
			armies[ownerNames[owner]] = newArmies
		} else {
			armies[ownerNames[owner]] = newArmies
		}
	}
	return armies
}

func (state *MapTileActor) getCityPID() (*actor.PID, error) {
	// can cities ever be removed? if so this should not use sync.Once
	state.cityOnce.Do(func() {
		getCityPIDResponse, err := Request[messages.GetCityPIDResponseMessage](system.Root, GetManagerPID(), messages.GetCityPIDMessage{
			CityId: state.Tile.CityId,
		})
		if err != nil {
			log.Printf("Error retreiving city pid: %s", err)
			return
		}
		if getCityPIDResponse.PID == nil {
			return
		}
		state.CityPID = getCityPIDResponse.PID
	})
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
		return nil, nil
	}
	return getBuildingPIDResponse.PID, nil
}
