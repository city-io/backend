package services

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"cityio/internal/constants"
	"cityio/internal/contracts"
	"cityio/internal/domain"
	"cityio/internal/logger"
	"cityio/internal/messages"
)

func RestoreCity(ctx context.Context, cluster contracts.ClusterProvider, city *domain.City) error {
	if _, err := cluster.Request("city", city.CityID, &messages.CreateCityMessage{City: *city, Restore: true}); err != nil {
		slog.ErrorContext(ctx, "failed to restore city actor", "city_id", city.CityID, "error", err)
		return err
	}

	return nil
}

func CreateCity(ctx context.Context, cluster contracts.ClusterProvider, store contracts.Store, world contracts.WorldProvider, city *CityInput) (*domain.City, error) {
	cityID := uuid.New().String()
	ctx = logger.With(ctx, "city_id", cityID)
	slog.InfoContext(ctx, "creating new city actor", "name", city.Name)

	block, err := world.ReserveCity(city.Size)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch empty city block", "error", err)
		return nil, err
	}

	startX := block.X
	startY := block.Y
	newCity := domain.City{
		CityID:          cityID,
		Type:            city.Type,
		Owner:           city.Owner,
		Name:            city.Name,
		Population:      constants.InitialPlayerCityPopulation,
		PopulationCap:   constants.InitialPlayerCityPopulation,
		GarrisonPercent: constants.DefaultGarrisonPercent,
		TaxRatePercent:  constants.DefaultTaxRatePercent,
		StartX:          startX,
		StartY:          startY,
		Size:            city.Size,
	}
	newCity.GarrisonPopulation = constants.GarrisonTarget(newCity)
	newCity.TaxIncomeRate = constants.TaxIncomePerHour(newCity)

	if _, err = cluster.Request("city", cityID, &messages.CreateCityMessage{City: newCity, Restore: false}); err != nil {
		slog.ErrorContext(ctx, "failed to create city actor", "error", err)
		return nil, err
	}

	return &newCity, nil
}
