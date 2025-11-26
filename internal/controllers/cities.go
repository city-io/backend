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
	c.log.Info("creating new city actor", "name", city.Name)
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
		Owner:         city.Owner,
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
		c.log.Error("failed to create city actor", "city_id", cityID, "error", err)
		return nil, err
	}

	return &newCity, nil
}
