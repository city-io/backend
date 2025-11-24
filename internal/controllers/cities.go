package controllers

import (
	"github.com/google/uuid"

	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
)

type cityController struct {
	cluster ports.ClusterProvider
	log     ports.Logger
}

func NewCityController(cl ports.ClusterProvider, l ports.Logger) ports.CityController {
	return &cityController{
		cluster: cl,
		log:     l,
	}
}

func (c *cityController) Restore(city *models.City) error {
	_, err := c.cluster.Request("city", city.CityID, &messages.CreateCityMessage{
		City:    *city,
		Restore: true,
	})
	if err != nil {
		c.log.Error("failed to restore city actor", "city_id", city.CityID, "error", err)
		return err
	}

	return nil
}

func (c *cityController) Create(city *models.CityInput) (*models.City, error) {
	cityID := uuid.New().String()

	tileFuture := c.cluster.RequestDBFuture(messages.GetEmptyCityBlockMessage{
		Size: constants.CitySize,
	})
	resp, err := tileFuture.Result()
	if err != nil {
		c.log.Error("failed to fetch empty city block", "error", err)
		return nil, err
	}
	randomTile := resp.(messages.GetEmptyCityBlockResponseMessage)

	startX := randomTile.X
	startY := randomTile.Y

	newCity := models.City{
		CityID:        cityID,
		Type:          city.Type,
		Owner:         &city.Owner,
		Name:          city.Name,
		Population:    constants.InitialPlayerCityPopulation,
		PopulationCap: constants.InitialPlayerCityPopulation,
		StartX:        startX,
		StartY:        startY,
		Size:          city.Size,
	}
	_, err = c.cluster.Request("city", cityID, &messages.CreateCityMessage{
		City:    newCity,
		Restore: false,
	})
	if err != nil {
		c.log.Error("Failed to create city actor", "city_id", cityID, "error", err)
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
