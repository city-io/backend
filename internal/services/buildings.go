package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"cityio/internal/contracts"
	"cityio/internal/domain"
	"cityio/internal/logger"
	"cityio/internal/messages"
)

func RestoreBuilding(ctx context.Context, cluster contracts.ClusterProvider, building *domain.Building) error {
	if _, err := cluster.Request("building", building.BuildingID, &messages.CreateBuildingMessage{Building: *building, Restore: true}); err != nil {
		slog.ErrorContext(ctx, "failed to restore building actor", "building_id", building.BuildingID, "error", err)
		return err
	}

	return nil
}

func CreateBuilding(ctx context.Context, cluster contracts.ClusterProvider, building *BuildingInput) (*domain.Building, error) {
	buildingID := uuid.New().String()
	ctx = logger.With(ctx, "building_id", buildingID)
	slog.InfoContext(ctx, "creating new building actor", "type", building.Type)

	newBuilding := domain.Building{
		BuildingID: buildingID,
		CityID:     building.CityID,
		Owner:      building.Owner,
		Type:       string(building.Type),
		X:          building.X,
		Y:          building.Y,
	}

	res, err := cluster.Request("building", buildingID, &messages.CreateBuildingMessage{Building: newBuilding, Restore: false, Construct: true})
	if err != nil {
		slog.ErrorContext(ctx, "failed to create building actor", "error", err)
		return nil, err
	}
	if _, ok := res.(messages.Ack); !ok {
		if responseErr, ok := res.(error); ok {
			return nil, responseErr
		}
		return nil, fmt.Errorf("unexpected building create response: %T", res)
	}
	res, err = cluster.Request("building", buildingID, messages.GetBuildingMessage{})
	if err != nil {
		return nil, err
	}
	response, ok := res.(*messages.GetBuildingResponseMessage)
	if !ok {
		return nil, fmt.Errorf("unexpected building response after create: %T", res)
	}
	return &response.Building, nil
}
