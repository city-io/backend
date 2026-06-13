package services

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/logger"
	"cityio/internal/messages"
	"cityio/internal/ports"
)

func RestoreCity(ctx context.Context, cluster ports.ClusterProvider, city *domain.City) error {
	if _, err := cluster.Request("city", city.CityID, &messages.CreateCityMessage{City: *city, Restore: true}); err != nil {
		slog.ErrorContext(ctx, "failed to restore city actor", "city_id", city.CityID, "error", err)
		return err
	}

	return nil
}

func CreateCity(ctx context.Context, cluster ports.ClusterProvider, city *CityInput) (*domain.City, error) {
	cityID := uuid.New().String()
	ctx = logger.With(ctx, "city_id", cityID)
	slog.InfoContext(ctx, "creating new city actor", "name", city.Name)

	tileFuture := cluster.RequestDBFuture(messages.GetEmptyCityBlockMessage{
		Size: constants.CitySize,
	})
	resp, err := tileFuture.Result()
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch empty city block", "error", err)
		return nil, err
	}
	randomTile := resp.(messages.GetEmptyCityBlockResponseMessage)

	startX := randomTile.X
	startY := randomTile.Y
	newCity := domain.City{
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
		slog.ErrorContext(ctx, "failed to create city actor", "error", err)
		return nil, err
	}

	return &newCity, nil
}
