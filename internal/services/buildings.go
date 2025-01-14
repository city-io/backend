package services

import (
	"cityio/internal/actors"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
)

func RestoreBuilding(building models.Building) error {
	// TODO: spawn corresponding actor of building type
	var err error
	var buildingPID *actor.PID
	switch building.Type {
	case models.BUILDING_TYPE_CENTER:
		return nil
	case models.BUILDING_TYPE_BARRACKS:
		buildingPID, err = actors.Spawn(&actors.BarracksActor{})
	case models.BUILDING_TYPE_HOUSE:
		return nil
	case models.BUILDING_TYPE_FARM:
		return nil
	case models.BUILDING_TYPE_MINE:
		return nil
	default:
		return &messages.BuildingTypeNotFoundError{
			BuildingType: building.Type,
		}
	}

	if err != nil {
		log.Printf("Error spawning building actor: %s", err)
		return err
	}

	var createBuildingResponse *messages.CreateBuildingResponseMessage
	createBuildingResponse, err = actors.Request[messages.CreateBuildingResponseMessage](system.Root, buildingPID, &messages.CreateBuildingMessage{
		Building: building,
		Restore:  true,
	})
	if err != nil {
		log.Printf("Error creating building: %s", err)
		return err
	}
	if createBuildingResponse.Error != nil {
		log.Printf("Error creating building: %s", createBuildingResponse.Error)
		return createBuildingResponse.Error
	}

	var addBuildingPIDResponse *messages.AddBuildingPIDResponseMessage
	addBuildingPIDResponse, err = actors.Request[messages.AddBuildingPIDResponseMessage](system.Root, actors.GetManagerPID(), &messages.AddBuildingPIDMessage{
		BuildingId: building.BuildingId,
		PID:        buildingPID,
	})
	if err != nil {
		log.Printf("Error adding building pid: %s", err)
		return err
	}
	if addBuildingPIDResponse.Error != nil {
		log.Printf("Error adding building pid: %s", addBuildingPIDResponse.Error)
		return addBuildingPIDResponse.Error
	}

	var getMapTilePIDResponse *messages.GetMapTilePIDResponseMessage
	getMapTilePIDResponse, err = actors.Request[messages.GetMapTilePIDResponseMessage](system.Root, actors.GetManagerPID(), &messages.GetMapTilePIDMessage{
		X: building.X,
		Y: building.Y,
	})
	if err != nil {
		log.Printf("Error getting map tile pid: %s", err)
		return err
	}
	if getMapTilePIDResponse.PID == nil {
		log.Printf("Error getting map tile pid: %s", err)
		return &messages.MapTileNotFoundError{
			X: building.X,
			Y: building.Y,
		}
	}

	system.Root.Send(getMapTilePIDResponse.PID, &messages.UpdateTileBuildingPIDMessage{
		BuildingPID: buildingPID,
	})

	system.Root.Send(buildingPID, &messages.UpdateBuildingTilePIDMessage{
		TilePID: getMapTilePIDResponse.PID,
	})

	return nil
}
