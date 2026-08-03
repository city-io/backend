package messages

import (
	"fmt"

	"cityio/internal/domain"
)

// CreateArmyMessage spawns (or restores) an army actor. Restore skips the
// persistence create so a boot-time restore doesn't re-write existing rows.
type CreateArmyMessage struct {
	Army    domain.Army
	Restore bool
}

type GetArmyMessage struct{}
type GetArmyResponseMessage struct {
	Army domain.Army
}

// MoveArmyMessage sets an army's marching destination. The army steps one tile
// toward it each movement tick until it arrives.
type MoveArmyMessage struct {
	X int
	Y int
}

// MergeArmiesMessage folds the source army's troops into the target (the
// recipient of this message). Both must share an owner and a tile.
type MergeArmiesMessage struct {
	SourceArmyID string
}

// SurrenderTroopsMessage asks an army to hand over all its troops and shut
// down. Used by MergeArmies on the source army.
type SurrenderTroopsMessage struct{}
type SurrenderTroopsResponseMessage struct {
	Troops map[domain.TroopType]int64
}

type DeleteArmyMessage struct{}

// TrainTroopsMessage orders a barracks to train a batch of troops.
type TrainTroopsMessage struct {
	Type  domain.TroopType
	Count int64
}

// ReserveMilitaryPopulationMessage asks a city to reserve Count of its
// population as military. The city acks if it stays within the military cap,
// otherwise replies InsufficientPopulationError.
type ReserveMilitaryPopulationMessage struct {
	Count int64
}

// ReleaseMilitaryPopulationMessage returns a previously reserved military
// population to the civilian pool (used to roll back a failed training order).
type ReleaseMilitaryPopulationMessage struct {
	Count int64
}

// SetArmyUpkeepMessage reports an army's food upkeep to the city currently
// bearing it. Keyed by army so resends are idempotent.
type SetArmyUpkeepMessage struct {
	ArmyID string
	Amount int64
}

// RemoveArmyUpkeepMessage drops an army's upkeep contribution from a city (the
// army moved to a different nearest city, merged, or was destroyed).
type RemoveArmyUpkeepMessage struct {
	ArmyID string
}

// Errors
type InsufficientPopulationError struct {
	Available int64
	Requested int64
}

func (e *InsufficientPopulationError) Error() string {
	return fmt.Sprintf("insufficient trainable population: requested %d, available %d", e.Requested, e.Available)
}

type TrainingCapacityExceededError struct {
	Requested int64
	Capacity  int64
}

func (e *TrainingCapacityExceededError) Error() string {
	return fmt.Sprintf("training batch exceeds barracks capacity: requested %d, capacity %d", e.Requested, e.Capacity)
}

type InvalidTroopCountError struct {
	Count int64
}

func (e *InvalidTroopCountError) Error() string {
	return fmt.Sprintf("invalid troop count: %d", e.Count)
}

type ArmyNotFoundError struct {
	ArmyID string
}

func (e *ArmyNotFoundError) Error() string {
	return fmt.Sprintf("army not found: %s", e.ArmyID)
}
