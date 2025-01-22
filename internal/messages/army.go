package messages

import (
	"cityio/internal/models"

	"fmt"
)

type CreateArmyMessage struct {
	Army    models.Army
	Restore bool
}
type GetArmyMessage struct{}
type UpdateArmyMessage struct {
	Army models.Army
}
type DeleteArmyMessage struct {
	ArmyId string
}
type StartArmyMarchMessage struct {
	X int
	Y int
}
type UpdateArmyTileMessage struct{}

type CreateArmyResponseMessage struct {
	Error error
}
type GetArmyResponseMessage struct {
	Army models.Army
}
type UpdateArmyResponseMessage struct {
	Error error
}
type DeleteArmyResponseMessage struct {
	Error error
}

// Errors
type ArmyNotFoundError struct {
	ArmyId string
}

func (e *ArmyNotFoundError) Error() string {
	return fmt.Sprintf("Army not found: %s", e.ArmyId)
}
