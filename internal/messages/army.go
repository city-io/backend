package messages

import (
	"cityio/internal/models"

	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

type CreateArmyMessage struct {
	Army     models.Army
	OwnerPID *actor.PID
	Restore  bool
}
type GetArmyMessage struct{}
type DeleteArmyMessage struct {
	ArmyId string
}
type RestoreTrainingMessage struct {
	Training models.Training
}

type CreateArmyResponseMessage struct {
	Error error
}
type GetArmyResponseMessage struct {
	Army models.Army
}
type DeleteArmyResponseMessage struct {
	Error error
}
type RestoreTrainingResponseMessage struct {
	Error error
}

// Errors
type ArmyNotFoundError struct {
	ArmyId string
}

func (e *ArmyNotFoundError) Error() string {
	return fmt.Sprintf("Army not found: %s", e.ArmyId)
}
