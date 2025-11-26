package controllers

import (
	"github.com/google/uuid"

	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
)

type buildingController struct {
	cluster ports.ClusterProvider
	log     ports.Logger
}

func NewBuildingController(cl ports.ClusterProvider, l ports.Logger) ports.BuildingController {
	return &buildingController{
		cluster: cl,
		log:     l,
	}
}

func (b *buildingController) Restore(building *models.Building) error {
	_, err := b.cluster.Request("building", building.BuildingID, &messages.CreateBuildingMessage{
		Building: *building,
		Restore:  true,
	})
	if err != nil {
		b.log.Error("failed to restore building actor", "building_id", building.BuildingID, "error", err)
		return err
	}

	return nil
}

func (b *buildingController) Create(building *models.Building) (*models.Building, error) {
	b.log.Info("creating new building actor", "type", building.Type)

	newBuilding := *building
	newBuilding.BuildingID = uuid.New().String()
	_, err := b.cluster.Request("building", newBuilding.BuildingID, &messages.CreateBuildingMessage{
		Building:  newBuilding,
		Restore:   false,
		Construct: true,
	})
	if err != nil {
		b.log.Error("failed to create building actor", "building_id", building.BuildingID, "error", err)
		return nil, err
	}

	return &newBuilding, nil
}
