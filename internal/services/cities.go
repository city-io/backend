package services

import (
	"log"
	"math/rand"

	"github.com/google/uuid"

	"cityio/internal/constants"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
)

func RestoreCity(cl ports.ClusterProvider, city models.City) error {
	_, err := cl.Request(city.CityId, "city", &messages.CreateCityMessage{
		City:    city,
		Restore: true,
	})
	if err != nil {
		log.Println("Failed to restore city actor:", err)
		return err
	}

	return nil
}

func CreateCity(cl ports.ClusterProvider, city models.CityInput) (*models.City, error) {
	db := database.GetDB()

	cityID := uuid.New().String()

	var tiles []models.MapTile
	err := db.Raw(`
		SELECT x, y, city_id
		FROM map_tiles
		WHERE city_id = ''
		  AND NOT EXISTS (
			SELECT 1
			FROM map_tiles t2
			WHERE t2.x BETWEEN map_tiles.x AND map_tiles.x + ?
			  AND t2.y BETWEEN map_tiles.y AND map_tiles.y + ?
			  AND t2.city_id != ''
		  )
	`, constants.CITY_SIZE, constants.CITY_SIZE).Scan(&tiles).Error
	// add limit to this query to spawn new users closer together
	// 10000 adds sufficient spacing

	if err != nil {
		log.Println("Failed to fetch map empty tiles:", err)
		return &models.City{}, err
	}
	randomTile := tiles[rand.Intn(len(tiles))]
	startX := randomTile.X
	startY := randomTile.Y

	newCity := models.City{
		CityId:        cityID,
		Type:          city.Type,
		Owner:         city.Owner,
		Name:          city.Name,
		Population:    constants.INITIAL_PLAYER_CITY_POPULATION,
		PopulationCap: constants.INITIAL_PLAYER_CITY_POPULATION,
		StartX:        startX,
		StartY:        startY,
		Size:          city.Size,
	}
	_, err = cl.Request(cityID, "city", &messages.CreateCityMessage{
		City:    newCity,
		Restore: true,
	})
	if err != nil {
		log.Println("Failed to create city actor:", err)
		return &models.City{}, err
	}

	return &newCity, nil
}

// func GetCity(cityId string) (models.City, error) {
// 	getCityPIDResponse, err := actors.Request[messages.GetCityPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetCityPIDMessage{
// 		CityId: cityId,
// 	})
// 	if err != nil {
// 		return models.City{}, err
// 	}
// 	if getCityPIDResponse.PID == nil {
// 		return models.City{}, &messages.CityNotFoundError{
// 			CityId: cityId,
// 		}
// 	}

// 	getCityResponse, err := actors.Request[messages.GetCityResponseMessage](system.Root, getCityPIDResponse.PID, messages.GetCityMessage{})
// 	if err != nil {
// 		return models.City{}, err
// 	}

// 	return getCityResponse.City, nil
// }

// func DeleteUserCity(userId string) error {
// 	db := database.GetDB()

// 	var city models.City
// 	err := db.Where("owner = ? AND type = 'city'", userId).First(&city).Error
// 	if err != nil {
// 		return err
// 	}

// 	getCityPIDResponse, err := actors.Request[messages.GetCityPIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetCityPIDMessage{
// 		CityId: city.CityId,
// 	})
// 	if err != nil {
// 		log.Printf("Error getting city pid: %s", err)
// 		return err
// 	}
// 	if getCityPIDResponse.PID == nil {
// 		return &messages.CityNotFoundError{
// 			CityId: city.CityId,
// 		}
// 	}

// 	deleteCityResponse, err := actors.Request[messages.DeleteCityResponseMessage](system.Root, getCityPIDResponse.PID, messages.DeleteCityMessage{})
// 	if err != nil {
// 		log.Printf("Error deleting city: %s", err)
// 		return err
// 	}
// 	if deleteCityResponse.Error != nil {
// 		log.Printf("Error deleting city: %s", deleteCityResponse.Error)
// 		return deleteCityResponse.Error
// 	}

// 	return nil
// }
