package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"math/rand"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
)

func RestoreCity(city models.City) error {
	cityActor := actors.CityActor{}
	cityPID, err := cityActor.Spawn()
	if err != nil {
		log.Printf("Error spawning city actor: %s", err)
		return err
	}

	tilePIDs := make(map[int]map[int]*actor.PID)
	for i := 0; i < city.Size; i++ {
		for j := 0; j < city.Size; j++ {
			response, err := actors.Request[messages.GetMapTilePIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetMapTilePIDMessage{
				X: city.StartX + i,
				Y: city.StartY + j,
			})
			if err != nil {
				log.Printf("Error getting map tile pid: %s", err)
				return err
			}

			if _, ok := tilePIDs[i]; !ok {
				tilePIDs[i] = make(map[int]*actor.PID)
			}
			tilePIDs[i][j] = response.PID
		}
	}

	var userPID *actor.PID
	if city.Owner != "" {
		response, err := actors.Request[messages.GetUserPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetUserPIDMessage{
			UserId: city.Owner,
		})
		if err != nil {
			log.Printf("Error getting user pid: %s", err)
			return err
		}
		userPID = response.PID
	}

	createCityResponse, err := actors.Request[messages.CreateCityResponseMessage](system.Root, cityPID, messages.CreateCityMessage{
		City:     city,
		TilePIDs: tilePIDs,
		OwnerPID: userPID,
		Restore:  true,
	})
	if err != nil {
		log.Printf("Error creating city: %s", err)
		return err
	}
	if createCityResponse.Error != nil {
		log.Printf("Error creating city: %s", createCityResponse.Error)
		return createCityResponse.Error
	}

	addCityPIDResponse, err := actors.Request[messages.AddCityPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.AddCityPIDMessage{
		CityId: city.CityId,
		PID:    cityPID,
	})
	if err != nil {
		log.Printf("Error adding city pid: %s", err)
		return err
	}
	if addCityPIDResponse.Error != nil {
		log.Printf("Error adding city pid: %s", addCityPIDResponse.Error)
		return addCityPIDResponse.Error
	}

	var buildings []models.Building
	err = database.GetDb().Where("city_id = ?", city.CityId).Find(&buildings).Error
	if err != nil {
		log.Printf("Failed to fetch buildings for city %s: %s", city.CityId, err.Error())
		return err
	}

	for _, building := range buildings {
		switch building.Type {
		case "center":
			continue
		case "barracks":
			continue
		case "house":
			continue
		case "farm":
			continue
		case "mine":
			continue
		}
	}

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

	userPID, exists := state.GetUserPID(city.Owner)
	if !exists {
		return "", &messages.UserNotFoundError{UserId: city.Owner}
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
		OwnerPID: userPID,
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

	log.Printf("Created new city at (%d, %d)", startX, startY)
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

func DeleteUserCity(userId string) error {
	db := database.GetDb()

	var city models.City
	err := db.Where("owner = ? AND type = 'city'", userId).First(&city).Error
	if err != nil {
		return err
	}

	cityPID, exists := state.GetCityPID(city.CityId)
	if !exists {
		return &messages.CityNotFoundError{CityId: city.CityId}
	}

	future := system.Root.RequestFuture(cityPID, messages.DeleteCityMessage{}, time.Second*2)
	result, err := future.Result()
	if err != nil {
		return err
	}

	response, ok := result.(messages.DeleteCityResponseMessage)
	if !ok {
		return &messages.InternalError{}
	}

	if response.Error != nil {
		return response.Error
	}

	return nil
}
