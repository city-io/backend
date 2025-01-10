package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/state"

	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
)

func RestoreCity(city models.City) {
	log.Printf("Restoring city: %s", city.CityId)
	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewCityActor(database.GetDb())
	})
	newPID := system.Root.Spawn(props)
	system.Root.Send(newPID, messages.CreateCityMessage{
		City:    city,
		Restore: true,
	})
	state.AddCityPID(city.CityId, newPID)
}

func CreateCity(city models.City) (string, error) {
	cityId := uuid.New().String()

	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewCityActor(database.GetDb())
	})
	newPID := system.Root.Spawn(props)
	city.CityId = cityId
	future := system.Root.RequestFuture(newPID, messages.CreateCityMessage{
		City:    city,
		Restore: false,
	}, time.Second*2)

	response, err := future.Result()
	if err != nil {
		return "", err
	}

	if response, ok := response.(messages.CreateCityResponseMessage); ok {
		if response.Error != nil {
			return "", response.Error
		}
	} else {
		return "", &messages.InternalError{}
	}

	state.AddCityPID(cityId, newPID)
	return cityId, nil
}

func GetCity(cityId string) (models.City, error) {
	cityPID, exists := state.GetCityPID(cityId)
	if !exists {
		return models.City{}, &messages.CityNotFoundError{CityId: cityId}
	}

	future := system.Root.RequestFuture(cityPID, messages.GetCityMessage{}, time.Second*2)
	response, err := future.Result()
	if err != nil {
		return models.City{}, err
	}

	city, ok := response.(models.City)
	if !ok {
		return models.City{}, &messages.CityNotFoundError{CityId: cityId}
	}

	return city, nil
}
