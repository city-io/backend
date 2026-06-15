package messages

import (
	"fmt"

	"cityio/internal/domain"
)

type CreateBuildingMessage struct {
	Building  domain.Building
	Restore   bool
	Construct bool
}
type UpgradeBuildingMessage struct{}

type GetBuildingMessage struct{}

type DeleteBuildingMessage struct {
	BuildingID string
}

// type TrainTroopsMessage struct {
// 	Training domain.Training
// }
// type RestoreTrainingMessage struct {
// 	Training domain.Training
// }
// type DeleteTrainingMessage struct {
// 	BarracksId string
// }

type GetBuildingResponseMessage struct {
	Building domain.Building
}

// BuildingStateChangedMessage is sent by a building to its city when its state
// changes meaningfully (construction start/complete, creation). The city relays
// it to the stream so the owning client sees the update.
type BuildingStateChangedMessage struct {
	Building domain.Building
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
