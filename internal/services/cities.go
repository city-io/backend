package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/state"

	"log"
	"math/rand"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
)

func RestoreCity(city models.City) error {
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
			} else {
				return &messages.MapTileNotFoundError{X: city.StartX + i, Y: city.StartY + j}
			}
		}
	}
	system.Root.Send(newPID, messages.CreateCityMessage{
		City:     city,
		TilePIDs: tilePIDs,
		Restore:  true,
	})
	state.AddCityPID(city.CityId, newPID)
	return nil
}

func CreateCity(city models.CityInput) (string, error) {
	db := database.GetDb()

	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewCityActor(db)
	})
	newPID := system.Root.Spawn(props)

	var tiles []models.MapTile
	err := db.Raw(`
		SELECT x, y, city_id
		FROM map_tiles
		WHERE city_id = ''
		  AND NOT EXISTS (
			SELECT 1
			FROM map_tiles t2
			WHERE t2.x BETWEEN map_tiles.x AND map_tiles.x + 2
			  AND t2.y BETWEEN map_tiles.y AND map_tiles.y + 2
			  AND t2.city_id != ''
		  )
	`).Scan(&tiles).Error
	// add limit to this query to spawn new users closer together
	// 10000 adds sufficient spacing

	if err != nil {
		log.Println("Failed to fetch map empty tiles:", err)
		return "", err
	}

	randomTile := tiles[rand.Intn(len(tiles))]
	startX := randomTile.X
	startY := randomTile.Y

	tilePIDS := make(map[int]map[int]*actor.PID)
	for i := 0; i < city.Size; i++ {
		for j := 0; j < city.Size; j++ {
			tilePID, exists := state.GetMapTilePID(startX+i, startX+j)
			if exists {
				if _, ok := tilePIDS[i]; !ok {
					tilePIDS[i] = make(map[int]*actor.PID)
				}
				tilePIDS[i][j] = tilePID
			} else {
				return "", &messages.MapTileNotFoundError{X: startX + i, Y: startY + j}
			}
		}
	}

	cityId := uuid.New().String()
	future := system.Root.RequestFuture(newPID, messages.CreateCityMessage{
		City: models.City{
			CityId:     cityId,
			Type:       city.Type,
			Owner:      city.Owner,
			Name:       city.Name,
			Population: city.Population,
			StartX:     startX,
			StartY:     startY,
			Size:       city.Size,
		},
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

	log.Printf("Created new city at %d, %d", startX, startY)
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
