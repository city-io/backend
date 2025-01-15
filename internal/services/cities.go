package services

import (
	"cityio/internal/actors"
	"cityio/internal/constants"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"math/rand"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
)

func RestoreCity(city models.City) error {
	cityPID, err := actors.Spawn(&actors.CityActor{})
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

	return nil
}

func CreateCity(city models.CityInput) (*models.City, error) {
	db := database.GetDb()

	cityPID, err := actors.Spawn(&actors.CityActor{})

	var tiles []models.MapTile
	err = db.Raw(`
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
		return &models.City{}, err
	}

	randomTile := tiles[rand.Intn(len(tiles))]
	startX := randomTile.X
	startY := randomTile.Y

	tilePIDS := make(map[int]map[int]*actor.PID)
	for i := 0; i < city.Size; i++ {
		for j := 0; j < city.Size; j++ {
			response, err := actors.Request[messages.GetMapTilePIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetMapTilePIDMessage{
				X: startX + i,
				Y: startY + j,
			})
			if err != nil {
				log.Printf("Error getting map tile pid: %s", err)
				return &models.City{}, err
			}
			if _, ok := tilePIDS[i]; !ok {
				tilePIDS[i] = make(map[int]*actor.PID)
			}
			tilePIDS[i][j] = response.PID
		}
	}

	response, err := actors.Request[messages.GetUserPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetUserPIDMessage{
		UserId: city.Owner,
	})
	if err != nil {
		log.Printf("Error getting user pid: %s", err)
		return &models.City{}, err
	}
	if response.PID == nil {
		return &models.City{}, &messages.UserNotFoundError{
			UserId: city.Owner,
		}
	}

	cityId := uuid.New().String()
	newCity := models.City{
		CityId:        cityId,
		Type:          city.Type,
		Owner:         city.Owner,
		Name:          city.Name,
		Population:    constants.INITIAL_PLAYER_CITY_POPULATION,
		PopulationCap: constants.INITIAL_PLAYER_CITY_POPULATION,
		StartX:        startX,
		StartY:        startY,
		Size:          city.Size,
	}

	var createCityResponse *messages.CreateCityResponseMessage
	createCityResponse, err = actors.Request[messages.CreateCityResponseMessage](system.Root, cityPID, messages.CreateCityMessage{
		City:     newCity,
		TilePIDs: tilePIDS,
		OwnerPID: response.PID,
		Restore:  false,
	})
	if err != nil {
		log.Printf("Error creating city: %s", err)
		return &models.City{}, err
	}
	if createCityResponse.Error != nil {
		log.Printf("Error creating city: %s", createCityResponse.Error)
		return &models.City{}, createCityResponse.Error
	}

	log.Printf("Created new city at (%d, %d)", startX, startY)

	addCityPIDResponse, err := actors.Request[messages.AddCityPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.AddCityPIDMessage{
		CityId: cityId,
		PID:    cityPID,
	})
	if err != nil {
		log.Printf("Error adding city pid: %s", err)
		return &models.City{}, err
	}
	if addCityPIDResponse.Error != nil {
		log.Printf("Error adding city pid: %s", addCityPIDResponse.Error)
		return &models.City{}, addCityPIDResponse.Error
	}

	return &newCity, nil
}

func GetCity(cityId string) (models.City, error) {
	getCityPIDResponse, err := actors.Request[messages.GetCityPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetCityPIDMessage{
		CityId: cityId,
	})
	if err != nil {
		return models.City{}, err
	}
	if getCityPIDResponse.PID == nil {
		return models.City{}, &messages.CityNotFoundError{
			CityId: cityId,
		}
	}

	getCityResponse, err := actors.Request[messages.GetCityResponseMessage](system.Root, getCityPIDResponse.PID, messages.GetCityMessage{})
	if err != nil {
		return models.City{}, err
	}

	return getCityResponse.City, nil
}

func DeleteUserCity(userId string) error {
	db := database.GetDb()

	var city models.City
	err := db.Where("owner = ? AND type = 'city'", userId).First(&city).Error
	if err != nil {
		return err
	}

	getCityPIDResponse, err := actors.Request[messages.GetCityPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetCityPIDMessage{
		CityId: city.CityId,
	})
	if err != nil {
		log.Printf("Error getting city pid: %s", err)
		return err
	}
	if getCityPIDResponse.PID == nil {
		return &messages.CityNotFoundError{
			CityId: city.CityId,
		}
	}

	deleteCityResponse, err := actors.Request[messages.DeleteCityResponseMessage](system.Root, getCityPIDResponse.PID, messages.DeleteCityMessage{})
	if err != nil {
		log.Printf("Error deleting city: %s", err)
		return err
	}
	if deleteCityResponse.Error != nil {
		log.Printf("Error deleting city: %s", deleteCityResponse.Error)
		return deleteCityResponse.Error
	}

	return nil
}
