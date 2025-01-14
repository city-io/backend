package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
)

func RestoreArmy(army models.Army) error {
	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewArmyActor(database.GetDb())
	})
	newPID := system.Root.Spawn(props)

	userPID, exists := state.GetUserPID(army.Owner)
	if !exists {
		return &messages.UserNotFoundError{UserId: army.Owner}
	}

	future := system.Root.RequestFuture(newPID, messages.CreateArmyMessage{
		Army:     army,
		OwnerPID: userPID,
		Restore:  true,
	}, time.Second*2)
	response, err := future.Result()
	if err != nil {
		return err
	}

	if response, ok := response.(messages.CreateArmyResponseMessage); ok {
		if response.Error != nil {
			return response.Error
		}
	} else {
		return &messages.InternalError{}
	}

	future = system.Root.RequestFuture(userPID, messages.AddUserArmyMessage{
		ArmyId:  army.ArmyId,
		ArmyPID: newPID,
	}, time.Second*2)

	response, err = future.Result()
	if err != nil {
		return err
	}

	if response, ok := response.(messages.AddUserArmyResponseMessage); ok {
		if response.Error != nil {
			return response.Error
		}
	} else {
		return &messages.InternalError{}
	}

	log.Printf("Restored army at (%d, %d)", army.TileX, army.TileY)

	state.AddArmyPID(army.ArmyId, newPID)
	return nil
}

func CreateArmy(army models.Army) (string, error) {
	db := database.GetDb()

	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewArmyActor(db)
	})
	newPID := system.Root.Spawn(props)

	userPID, exists := state.GetUserPID(army.Owner)
	if !exists {
		return "", &messages.UserNotFoundError{UserId: army.Owner}
	}
	army.ArmyId = uuid.New().String()
	future := system.Root.RequestFuture(newPID, messages.CreateArmyMessage{
		Army:     army,
		OwnerPID: userPID,
		Restore:  false,
	}, time.Second*2)

	response, err := future.Result()
	if err != nil {
		return "", err
	}

	if response, ok := response.(messages.CreateArmyResponseMessage); ok {
		if response.Error != nil {
			return "", response.Error
		}
	} else {
		return "", &messages.InternalError{}
	}

	future = system.Root.RequestFuture(userPID, messages.AddUserArmyMessage{
		ArmyId:  army.ArmyId,
		ArmyPID: newPID,
	}, time.Second*2)

	response, err = future.Result()
	if err != nil {
		return "", err
	}

	if response, ok := response.(messages.AddUserArmyResponseMessage); ok {
		if response.Error != nil {
			return "", response.Error
		}
	} else {
		return "", &messages.InternalError{}
	}

	log.Printf("Created new army at (%d, %d)", army.TileX, army.TileY)
	state.AddArmyPID(army.ArmyId, newPID)
	return army.ArmyId, nil
}

func GetArmy(armyId string) (models.Army, error) {
	armyPID, exists := state.GetArmyPID(armyId)
	if !exists {
		return models.Army{}, &messages.ArmyNotFoundError{ArmyId: armyId}
	}

	future := system.Root.RequestFuture(armyPID, messages.GetArmyMessage{}, time.Second*2)
	result, err := future.Result()
	if err != nil {
		return models.Army{}, err
	}

	response, ok := result.(messages.GetArmyResponseMessage)
	if !ok {
		return models.Army{}, &messages.ArmyNotFoundError{ArmyId: armyId}
	}

	return response.Army, nil
}

func DeleteUserArmies(userId string) error {
	db := database.GetDb()

	var armies []models.Army
	err := db.Where("owner = ?", userId).Find(&armies).Error
	if err != nil {
		return err
	}

	for _, army := range armies {
		armyPID, exists := state.GetArmyPID(army.ArmyId)
		if !exists {
			return &messages.ArmyNotFoundError{ArmyId: army.ArmyId}
		}

		future := system.Root.RequestFuture(armyPID, messages.DeleteArmyMessage{}, time.Second*2)
		result, err := future.Result()
		if err != nil {
			return err
		}

		response, ok := result.(messages.DeleteArmyResponseMessage)
		if !ok {
			return &messages.InternalError{}
		}

		if response.Error != nil {
			return response.Error
		}
	}

	return nil
}
