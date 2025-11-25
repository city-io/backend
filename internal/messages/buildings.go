package messages

import (
	"fmt"

	"cityio/internal/models"
)

type CreateBuildingMessage struct {
	Building models.Building
	Restore  bool
}
type UpgradeBuildingMessage struct{}

type GetBuildingMessage struct{}

type UpdateBuildingOwnerMessage struct {
	Owner string
}
type UpdateBuildingMessage struct {
	Building models.Building
}
type DeleteBuildingMessage struct {
	BuildingID string
}

// type TrainTroopsMessage struct {
// 	Training models.Training
// }
// type RestoreTrainingMessage struct {
// 	Training models.Training
// }
// type DeleteTrainingMessage struct {
// 	BarracksId string
// }

type GetBuildingResponseMessage struct {
	Building models.Building
}

// // Errors
// type BuildingTypeNotFoundError struct {
// 	BuildingType string
// }

// func (e *BuildingTypeNotFoundError) Error() string {
// 	return fmt.Sprintf("Building type not found: %s", e.BuildingType)
// }

// type BuildingNotFoundError struct {
// 	BuildingId string
// }

// func (e *BuildingNotFoundError) Error() string {
// 	return fmt.Sprintf("Building not found: %s", e.BuildingId)
// }

// type TrainingAlreadyExistsError struct {
// 	BarracksId string
// }

// func (e *TrainingAlreadyExistsError) Error() string {
// 	return fmt.Sprintf("Training already exists for barracks: %s", e.BarracksId)
// }

type ConstructionInProgressError struct {
	BuildingID string
}

func (e *ConstructionInProgressError) Error() string {
	return fmt.Sprintf("Construction already active for building: %s", e.BuildingID)
}

type MaxLevelReachedError struct {
	BuildingID string
}

func (e *MaxLevelReachedError) Error() string {
	return fmt.Sprintf("Max level reached for building: %s", e.BuildingID)
}
