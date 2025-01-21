package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/google/uuid"
)

func RestoreArmy(army models.Army) error {
	armyPID, err := actors.Spawn(&actors.ArmyActor{})
	if err != nil {
		log.Printf("Error spawning actor for restored army: %s", err)
		return err
	}

	getUserPIDResponse, err := actors.Request[messages.GetUserPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetUserPIDMessage{
		UserId: army.Owner,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return err
	}
	if getUserPIDResponse.PID == nil {
		log.Printf("Error restoring army: User not found")
		return &messages.UserNotFoundError{UserId: army.Owner}
	}

	createArmyResponse, err := actors.Request[messages.CreateArmyResponseMessage](system.Root, armyPID, messages.CreateArmyMessage{
		Army:     army,
		OwnerPID: getUserPIDResponse.PID,
		Restore:  true,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return err
	}
	if createArmyResponse.Error != nil {
		log.Printf("Error restoring army: %s", createArmyResponse.Error)
		return createArmyResponse.Error
	}

	addUserArmyResponse, err := actors.Request[messages.AddUserArmyResponseMessage](system.Root, getUserPIDResponse.PID, messages.AddUserArmyMessage{
		ArmyId:  army.ArmyId,
		ArmyPID: armyPID,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return err
	}
	if addUserArmyResponse.Error != nil {
		log.Printf("Error restoring army: %s", addUserArmyResponse.Error)
		return addUserArmyResponse.Error
	}

	getTilePIDResponse, err := actors.Request[messages.GetMapTilePIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetMapTilePIDMessage{
		X: army.TileX,
		Y: army.TileY,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return err
	}
	if getTilePIDResponse.PID == nil {
		log.Printf("Error restoring army: Map tile not found")
		return &messages.MapTileNotFoundError{X: army.TileX, Y: army.TileY}
	}

	// TODO: replace with better way of storing armies in tiles
	addTileArmyPIDResponse, err := actors.Request[messages.AddTileArmyResponseMessage](system.Root, getTilePIDResponse.PID, messages.AddTileArmyMessage{
		ArmyPID: armyPID,
		Army:    army,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return err
	}
	if addTileArmyPIDResponse.Error != nil {
		log.Printf("Error restoring army: %s", addTileArmyPIDResponse.Error)
		return addTileArmyPIDResponse.Error
	}

	addArmyPIDResponse, err := actors.Request[messages.AddArmyPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.AddArmyPIDMessage{
		ArmyId: army.ArmyId,
		PID:    armyPID,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return err
	}
	if addArmyPIDResponse.Error != nil {
		log.Printf("Error restoring army: %s", addArmyPIDResponse.Error)
		return addArmyPIDResponse.Error
	}

	log.Printf("Restored army at (%d, %d) of size %d", army.TileX, army.TileY, army.Size)
	return nil
}

func CreateArmy(army models.Army) (string, error) {
	armyPID, err := actors.Spawn(&actors.ArmyActor{})

	getUserPIDResponse, err := actors.Request[messages.GetUserPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetUserPIDMessage{
		UserId: army.Owner,
	})
	if err != nil {
		log.Printf("Error creating army: %s", err)
		return "", err
	}
	if getUserPIDResponse.PID == nil {
		log.Printf("Error creating army: User not found")
		return "", &messages.UserNotFoundError{UserId: army.Owner}
	}

	army.ArmyId = uuid.New().String()
	createArmyResponse, err := actors.Request[messages.CreateArmyResponseMessage](system.Root, armyPID, messages.CreateArmyMessage{
		Army:     army,
		OwnerPID: getUserPIDResponse.PID,
		Restore:  false,
	})
	if err != nil {
		log.Printf("Error creating army: %s", err)
		return "", err
	}
	if createArmyResponse.Error != nil {
		log.Printf("Error creating army: %s", createArmyResponse.Error)
		return "", createArmyResponse.Error
	}

	addUserArmyResponse, err := actors.Request[messages.AddUserArmyResponseMessage](system.Root, getUserPIDResponse.PID, messages.AddUserArmyMessage{
		ArmyId:  army.ArmyId,
		ArmyPID: armyPID,
	})
	if err != nil {
		log.Printf("Error creating army: %s", err)
		return "", err
	}
	if addUserArmyResponse.Error != nil {
		log.Printf("Error creating army: %s", addUserArmyResponse.Error)
		return "", addUserArmyResponse.Error
	}

	getTilePIDResponse, err := actors.Request[messages.GetMapTilePIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetMapTilePIDMessage{
		X: army.TileX,
		Y: army.TileY,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return "", err
	}
	if getTilePIDResponse.PID == nil {
		log.Printf("Error restoring army: Map tile not found")
		return "", &messages.MapTileNotFoundError{X: army.TileX, Y: army.TileY}
	}

	// TODO: replace with better way of storing armies in tiles
	addTileArmyPIDResponse, err := actors.Request[messages.AddTileArmyResponseMessage](system.Root, getTilePIDResponse.PID, messages.AddTileArmyMessage{
		ArmyPID: armyPID,
		Army:    army,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return "", err
	}
	if addTileArmyPIDResponse.Error != nil {
		log.Printf("Error restoring army: %s", addTileArmyPIDResponse.Error)
		return "", addTileArmyPIDResponse.Error
	}

	addArmyPIDResponse, err := actors.Request[messages.AddArmyPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.AddArmyPIDMessage{
		ArmyId: army.ArmyId,
		PID:    armyPID,
	})
	if err != nil {
		log.Printf("Error creating army: %s", err)
		return "", err
	}
	if addArmyPIDResponse.Error != nil {
		log.Printf("Error creating army: %s", addArmyPIDResponse.Error)
		return "", addArmyPIDResponse.Error
	}

	log.Printf("Created army at (%d, %d) of size %d", army.TileX, army.TileY, army.Size)
	return army.ArmyId, nil
}

func GetArmy(armyId string) (models.Army, error) {
	getArmyPIDResponse, err := actors.Request[messages.GetArmyPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetArmyPIDMessage{
		ArmyId: armyId,
	})
	if err != nil {
		log.Printf("Error getting army: %s", err)
		return models.Army{}, err
	}
	if getArmyPIDResponse.PID == nil {
		return models.Army{}, &messages.ArmyNotFoundError{ArmyId: armyId}
	}

	getArmyResponse, err := actors.Request[messages.GetArmyResponseMessage](system.Root, getArmyPIDResponse.PID, messages.GetArmyMessage{})
	if err != nil {
		log.Printf("Error getting army: %s", err)
		return models.Army{}, err
	}

	return getArmyResponse.Army, nil
}

func DeleteUserArmies(userId string) error {
	db := database.GetDb()

	var armies []models.Army
	err := db.Where("owner = ?", userId).Find(&armies).Error
	if err != nil {
		return err
	}

	for _, army := range armies {
		getArmyPIDResponse, err := actors.Request[messages.GetArmyPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetArmyPIDMessage{
			ArmyId: army.ArmyId,
		})
		if err != nil {
			log.Printf("Error deleting user armies: %s", err)
			return err
		}
		if getArmyPIDResponse.PID == nil {
			return &messages.ArmyNotFoundError{ArmyId: army.ArmyId}
		}

		deleteArmyResponse, err := actors.Request[messages.DeleteArmyResponseMessage](system.Root, getArmyPIDResponse.PID, messages.DeleteArmyMessage{})
		if err != nil {
			log.Printf("Error deleting user armies: %s", err)
			return err
		}
		if deleteArmyResponse.Error != nil {
			log.Printf("Error deleting user armies: %s", deleteArmyResponse.Error)
			return deleteArmyResponse.Error
		}

		deleteArmyPIDResponse, err := actors.Request[messages.DeleteArmyPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.DeleteArmyPIDMessage{
			ArmyId: army.ArmyId,
		})
		if err != nil {
			log.Printf("Error deleting user armies: %s", err)
			return err
		}
		if deleteArmyPIDResponse.Error != nil {
			log.Printf("Error deleting user armies: %s", deleteArmyPIDResponse.Error)
			return deleteArmyPIDResponse.Error
		}
	}

	return nil
}
