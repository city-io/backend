package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/state"

	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
)

func RestoreCity(city models.City) {
	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewCityActor(database.GetDb())
	})
	newPID := system.Root.Spawn(props)

	tilePIDs := make(map[int]map[int]*actor.PID)
	for i := 0; i < city.Size; i++ {
		for j := 0; j < city.Size; j++ {
			tilePID, exists := state.GetMapTilePID(city.StartX+i, city.StartY+j)
			if exists {
				if _, ok := tilePIDs[i]; !ok {
					tilePIDs[i] = make(map[int]*actor.PID)
				}
				tilePIDs[i][j] = tilePID
			}
		}
	}
	system.Root.Send(newPID, messages.CreateCityMessage{
		City:     city,
		TilePIDs: tilePIDs,
		Restore:  true,
	})
	state.AddCityPID(city.CityId, newPID)
}

func CreateCity(city models.City) (string, error) {
	cityId := uuid.New().String()

	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewCityActor(database.GetDb())
	})
	newPID := system.Root.Spawn(props)

	tilePIDS := make(map[int]map[int]*actor.PID)
	for i := 0; i < city.Size; i++ {
		for j := 0; j < city.Size; j++ {
			tilePID, exists := state.GetMapTilePID(city.StartX+i, city.StartX+j)
			if exists {
				if _, ok := tilePIDS[i]; !ok {
					tilePIDS[i] = make(map[int]*actor.PID)
				}
				tilePIDS[i][j] = tilePID
			}
		}
	}

	city.CityId = cityId
	future := system.Root.RequestFuture(newPID, messages.CreateCityMessage{
		City:     city,
		TilePIDs: tilePIDS,
		Restore:  false,
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
	result, err := future.Result()
	if err != nil {
		return models.City{}, err
	}

	response, ok := result.(messages.GetCityResponseMessage)
	if !ok {
		return models.City{}, &messages.CityNotFoundError{CityId: cityId}
	}

	return response.City, nil
}
