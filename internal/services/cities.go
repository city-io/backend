package services

import (
	"context"

	"github.com/google/uuid"

	"cityio/internal/constants"
	"cityio/internal/logger"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
)

func RestoreCity(ctx context.Context, cluster ports.ClusterProvider, city *models.City) error {
	if _, err := cluster.Request("city", city.CityID, &messages.CreateCityMessage{City: *city, Restore: true}); err != nil {
		if log := logger.FromContext(ctx); log != nil {
			log.Error("failed to restore city actor", "city_id", city.CityID, "error", err)
		}
		return err
	}

	return nil
}

func CreateCity(ctx context.Context, cluster ports.ClusterProvider, city *models.CityInput) (*models.City, error) {
	log := logger.FromContext(ctx)
	if log != nil {
		log.Info("creating new city actor", "name", city.Name)
	}

	cityID := uuid.New().String()

	tileFuture := cluster.RequestDBFuture(messages.GetEmptyCityBlockMessage{
		Size: constants.CitySize,
	})
	resp, err := tileFuture.Result()
	if err != nil {
		if log != nil {
			log.Error("failed to fetch empty city block", "error", err)
		}
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

	if _, err = cluster.Request("city", cityID, &messages.CreateCityMessage{City: newCity, Restore: false}); err != nil {
		if log != nil {
			log.Error("failed to create city actor", "city_id", cityID, "error", err)
		}
		return nil, err
	}

	return &newCity, nil
}
