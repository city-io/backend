package services

import (
	"cityio/internal/actors"
	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"

	"errors"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func RestoreBuilding(building models.Building) error {
	var err error
	var buildingPID *actor.PID
	switch building.Type {
	case constants.BUILDING_TYPE_CITY_CENTER:
		buildingPID, err = actors.Spawn(&actors.CityCenterActor{})
	case constants.BUILDING_TYPE_TOWN_CENTER:
		buildingPID, err = actors.Spawn(&actors.TownCenterActor{})
	case constants.BUILDING_TYPE_BARRACKS:
		buildingPID, err = actors.Spawn(&actors.BarracksActor{})
	case constants.BUILDING_TYPE_HOUSE:
		buildingPID, err = actors.Spawn(&actors.HouseActor{})
	case constants.BUILDING_TYPE_FARM:
		buildingPID, err = actors.Spawn(&actors.FarmActor{})
	case constants.BUILDING_TYPE_MINE:
		buildingPID, err = actors.Spawn(&actors.MineActor{})
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
	createBuildingResponse, err = actors.Request[messages.CreateBuildingResponseMessage](system.Root, buildingPID, messages.CreateBuildingMessage{
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
	addBuildingPIDResponse, err = actors.Request[messages.AddBuildingPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.AddBuildingPIDMessage{
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

	if building.Type == constants.BUILDING_TYPE_BARRACKS {
		var training models.Training
		result := db.Where("barracks_id = ?", building.BuildingId).First(&training)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil
			}
			log.Printf("Error getting training: %s", result.Error)
			return result.Error
		}

		var restoreTrainingResponse *messages.RestoreTrainingResponseMessage
		restoreTrainingResponse, err = actors.Request[messages.RestoreTrainingResponseMessage](system.Root, buildingPID, messages.RestoreTrainingMessage{
			Training: training,
		})
		if err != nil {
			log.Printf("Error restoring training: %s", err)
			return err
		}
		if restoreTrainingResponse.Error != nil {
			log.Printf("Error restoring training: %s", restoreTrainingResponse.Error)
			return restoreTrainingResponse.Error
		}
	}

	return nil
}

func ConstructBuilding(building models.Building) (string, error) {
	var err error
	var buildingPID *actor.PID
	switch building.Type {
	case constants.BUILDING_TYPE_CITY_CENTER:
		buildingPID, err = actors.Spawn(&actors.CityCenterActor{})
	case constants.BUILDING_TYPE_TOWN_CENTER:
		buildingPID, err = actors.Spawn(&actors.TownCenterActor{})
	case constants.BUILDING_TYPE_BARRACKS:
		buildingPID, err = actors.Spawn(&actors.BarracksActor{})
	case constants.BUILDING_TYPE_HOUSE:
		buildingPID, err = actors.Spawn(&actors.HouseActor{})
	case constants.BUILDING_TYPE_FARM:
		buildingPID, err = actors.Spawn(&actors.FarmActor{})
	case constants.BUILDING_TYPE_MINE:
		buildingPID, err = actors.Spawn(&actors.MineActor{})
	default:
		return "", &messages.BuildingTypeNotFoundError{
			BuildingType: building.Type,
		}
	}

	if err != nil {
		log.Printf("Error spawning building actor: %s", err)
		return "", err
	}

	building.BuildingId = uuid.New().String()
	building.ConstructionEnd = time.Now().Add(time.Duration(constants.GetBuildingConstructionTime(building.Type, 1)) * time.Second)
	var createBuildingResponse *messages.CreateBuildingResponseMessage
	createBuildingResponse, err = actors.Request[messages.CreateBuildingResponseMessage](system.Root, buildingPID, messages.CreateBuildingMessage{
		Building: building,
		Restore:  false,
	})
	if err != nil {
		log.Printf("Error creating building: %s", err)
		return "", err
	}
	if createBuildingResponse.Error != nil {
		log.Printf("Error creating building: %s", createBuildingResponse.Error)
		return "", createBuildingResponse.Error
	}

	var addBuildingPIDResponse *messages.AddBuildingPIDResponseMessage
	addBuildingPIDResponse, err = actors.Request[messages.AddBuildingPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.AddBuildingPIDMessage{
		BuildingId: building.BuildingId,
		PID:        buildingPID,
	})
	if err != nil {
		log.Printf("Error adding building pid: %s", err)
		return "", err
	}
	if addBuildingPIDResponse.Error != nil {
		log.Printf("Error adding building pid: %s", addBuildingPIDResponse.Error)
		return "", addBuildingPIDResponse.Error
	}

	return building.BuildingId, nil
}

func TrainTroops(training models.Training) error {
	getBarracksPIDResponse, err := actors.Request[messages.GetBuildingPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetBuildingPIDMessage{
		BuildingId: training.BarracksId,
	})
	if err != nil {
		log.Printf("Error training troops: %s", err)
		return err
	}
	if getBarracksPIDResponse.PID == nil {
		log.Printf("Error training troops: %s", &messages.BuildingNotFoundError{BuildingId: training.BarracksId})
		return err
	}

	var trainResponse *messages.TrainTroopsResponseMessage
	trainResponse, err = actors.Request[messages.TrainTroopsResponseMessage](system.Root, getBarracksPIDResponse.PID, messages.TrainTroopsMessage{
		Training: training,
	})
	if err != nil {
		log.Printf("Error training troops: %s", err)
		return err
	}
	if trainResponse.Error != nil {
		log.Printf("Error training troops: %s", trainResponse.Error)
		return trainResponse.Error
	}

	return nil
}
