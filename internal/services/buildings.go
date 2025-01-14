package services

import (
	"cityio/internal/messages"
	"cityio/internal/models"
)

func RestoreBuilding(building models.Building) error {
	// TODO: spawn corresponding actor of building type
	switch building.Type {
	case "center":
		return nil
	case "barracks":
		return nil
	case "house":
		return nil
	case "farm":
		return nil
	case "mine":
		return nil
	default:
		return &messages.BuildingTypeNotFoundError{
			BuildingType: building.Type,
		}
	}
}
