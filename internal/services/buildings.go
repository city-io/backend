package services

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"cityio/internal/domain"
	"cityio/internal/logger"
	"cityio/internal/messages"
	"cityio/internal/ports"
)

func RestoreBuilding(ctx context.Context, cluster ports.ClusterProvider, building *domain.Building) error {
	if _, err := cluster.Request("building", building.BuildingID, &messages.CreateBuildingMessage{Building: *building, Restore: true}); err != nil {
		slog.ErrorContext(ctx, "failed to restore building actor", "building_id", building.BuildingID, "error", err)
		return err
	}

	return nil
}

func CreateBuilding(ctx context.Context, cluster ports.ClusterProvider, building *BuildingInput) (*domain.Building, error) {
	buildingID := uuid.New().String()
	ctx = logger.With(ctx, "building_id", buildingID)
	slog.InfoContext(ctx, "creating new building actor", "type", building.Type)

	newBuilding := domain.Building{
		BuildingID: buildingID,
		CityID:     building.CityID,
		Type:       string(building.Type),
		X:          building.X,
		Y:          building.Y,
	}

	if _, err := cluster.Request("building", buildingID, &messages.CreateBuildingMessage{Building: newBuilding, Restore: false, Construct: true}); err != nil {
		slog.ErrorContext(ctx, "failed to create building actor", "error", err)
		return nil, err
	}

	return &newBuilding, nil
}
