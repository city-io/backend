package services

import (
	"context"

	"github.com/google/uuid"

	"cityio/internal/logger"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
)

func RestoreBuilding(ctx context.Context, cluster ports.ClusterProvider, building *models.Building) error {
	if _, err := cluster.Request("building", building.BuildingID, &messages.CreateBuildingMessage{Building: *building, Restore: true}); err != nil {
		if log := logger.FromContext(ctx); log != nil {
			log.Error("failed to restore building actor", "building_id", building.BuildingID, "error", err)
		}
		return err
	}

	return nil
}

func CreateBuilding(ctx context.Context, cluster ports.ClusterProvider, building *models.BuildingInput) (*models.Building, error) {
	if log := logger.FromContext(ctx); log != nil {
		log.Info("creating new building actor", "type", building.Type)
	}

	buildingID := uuid.New().String()
	newBuilding := models.Building{
		BuildingID: buildingID,
		CityID:     building.CityID,
		Type:       string(building.Type),
		X:          building.X,
		Y:          building.Y,
	}

	if _, err := cluster.Request("building", buildingID, &messages.CreateBuildingMessage{Building: newBuilding, Restore: false, Construct: true}); err != nil {
		if log := logger.FromContext(ctx); log != nil {
			log.Error("failed to create building actor", "building_id", buildingID, "error", err)
		}
		return nil, err
	}

	return &newBuilding, nil
}
