package messages

import (
	"cityio/internal/models"

	"fmt"
)

type CreateBuildingMessage struct {
	Building models.Building
	Restore  bool
}
type UpgradeBuildingMessage struct{}
type GetBuildingMessage struct{}
type UpdateBuildingMessage struct {
	Building models.Building
}
type DeleteBuildingMessage struct {
	BuildingId string
}
type TrainTroopsMessage struct {
	Training models.Training
}
type RestoreTrainingMessage struct {
	Training models.Training
}
type DeleteTrainingMessage struct {
	BarracksId string
}

type CreateBuildingResponseMessage struct {
	Error error
}
type UpgradeBuildingResponseMessage struct {
	Error error
}
type GetBuildingResponseMessage struct {
	Building models.Building
}
type DeleteBuildingResponseMessage struct {
	Error error
}
type TrainTroopsResponseMessage struct {
	Error error
}
type RestoreTrainingResponseMessage struct {
	Error error
}

// Errors
type BuildingTypeNotFoundError struct {
	BuildingType string
}

func (e *BuildingTypeNotFoundError) Error() string {
	return fmt.Sprintf("Building type not found: %s", e.BuildingType)
}

type BuildingNotFoundError struct {
	BuildingId string
}

func (e *BuildingNotFoundError) Error() string {
	return fmt.Sprintf("Building not found: %s", e.BuildingId)
}

type TrainingAlreadyExistsError struct {
	BarracksId string
}

func (e *TrainingAlreadyExistsError) Error() string {
	return fmt.Sprintf("Training already exists for barracks: %s", e.BarracksId)
}

type MaxLevelReachedError struct {
	BuildingId string
}

func (e *MaxLevelReachedError) Error() string {
	return fmt.Sprintf("Max level reached for building: %s", e.BuildingId)
}
