package messages

import (
	"fmt"

	"cityio/internal/domain"
)

// CreateArmyMessage spawns (or restores) an army actor. Restore skips the
// persistence create so a boot-time restore doesn't re-write existing rows.
// SuppressPublish lets a compound operation publish only after every involved
// army has reached its final state.
type CreateArmyMessage struct {
	Army            domain.Army
	Restore         bool
	SuppressPublish bool
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

type AttackArmyMessage struct{ TargetArmyID string }
type ConquerSettlementMessage struct{ CityID string }
type RetreatArmyMessage struct{}

type EnterBattleMessage struct{ BattleID string }
type LeaveBattleMessage struct{ BattleID string }
type ApplyCasualtiesMessage struct{ Casualties map[domain.TroopType]int64 }
type ApplyCasualtiesResponseMessage struct {
	ArmyID   string
	Survived bool
}

type CreateBattleMessage struct {
	Battle domain.Battle
	Armies []domain.Army
}
type JoinBattleMessage struct {
	Army                 domain.Army
	OpposesArmyID        string
	OpposesMilitiaCityID string
}

type GetBattleMessage struct{}
type GetBattleResponseMessage struct{ Battle domain.Battle }
type RetreatFromBattleMessage struct{ ArmyID string }

// MergeArmiesMessage folds the source army's troops into the target (the
// recipient of this message). Both must share an owner and a tile.
type MergeArmiesMessage struct {
	SourceArmyID string
}
type MergeArmiesResponseMessage struct {
	Army          domain.Army
	DeletedArmyID string
}

type SplitArmyMessage struct {
	Troops map[domain.TroopType]int64
}
type SplitArmyResponseMessage struct {
	Source domain.Army
	Army   domain.Army
}

// SurrenderTroopsMessage asks an army to hand over all its troops and shut
// down. Used by MergeArmies on the source army.
type SurrenderTroopsMessage struct{}
type SurrenderTroopsResponseMessage struct {
	Troops map[domain.TroopType]int64
}

type DeleteArmyMessage struct{}

// TrainTroopsMessage adds a paid batch to a city's shared FIFO pipeline.
type TrainTroopsMessage struct {
	Type  domain.TroopType
	Count int64
}

type TrainTroopsResponseMessage struct {
	Order domain.TrainingOrder
}

type GetTrainingOrdersMessage struct{}
type GetTrainingOrdersResponseMessage struct {
	Orders []domain.TrainingOrder
}

type CancelTrainingOrderMessage struct {
	TrainingOrderID string
}

// ClaimTrainingOrderMessage lets an idle, completed barracks claim the oldest
// queued city order. Repeating the request returns its existing active order.
type ClaimTrainingOrderMessage struct {
	BarracksID string
	Level      int
}

type ClaimTrainingOrderResponseMessage struct {
	Order *domain.TrainingOrder
}

type CompleteTrainingOrderMessage struct {
	TrainingOrderID string
	BarracksID      string
}

type TrainingQueueAvailableMessage struct{}

// RecruitPopulationMessage transfers Count residents out of a city and into a
// durable training order. It rejects requests that would cross the protected
// civilian floor plus the city's configured militia target.
type RecruitPopulationMessage struct {
	Count int64
}

// ReturnRecruitsMessage restores residents when creation of their paid training
// order fails. It is a rollback path, not a casualty/disband mechanism.
type ReturnRecruitsMessage struct {
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

type TrainingInProgressError struct {
	BarracksID string
}

type TrainingAlreadyStartedError struct {
	TrainingOrderID string
}

func (e *TrainingAlreadyStartedError) Error() string {
	return fmt.Sprintf("training order has already started: %s", e.TrainingOrderID)
}

type TrainingOrderNotFoundError struct {
	TrainingOrderID string
}

type NoBarracksError struct{ CityID string }

func (e *NoBarracksError) Error() string {
	return fmt.Sprintf("city has no barracks: %s", e.CityID)
}

func (e *TrainingOrderNotFoundError) Error() string {
	return fmt.Sprintf("training order not found: %s", e.TrainingOrderID)
}

func (e *TrainingInProgressError) Error() string {
	return fmt.Sprintf("barracks has an active training order: %s", e.BarracksID)
}

type InvalidTroopCountError struct {
	Count int64
}

type InvalidTroopTypeError struct {
	Type domain.TroopType
}

func (e *InvalidTroopTypeError) Error() string {
	return fmt.Sprintf("invalid troop type: %s", e.Type)
}

func (e *InvalidTroopCountError) Error() string {
	return fmt.Sprintf("invalid troop count: %d", e.Count)
}

type ArmyNotFoundError struct {
	ArmyID string
}

type UnreachableDestinationError struct {
	X int
	Y int
}

type ArmyInBattleError struct{ ArmyID string }

func (e *ArmyInBattleError) Error() string { return fmt.Sprintf("army is in battle: %s", e.ArmyID) }

type InvalidArmySplitError struct{ Reason string }

func (e *InvalidArmySplitError) Error() string {
	return fmt.Sprintf("invalid army split: %s", e.Reason)
}

type InsufficientTroopsError struct {
	Type      domain.TroopType
	Available int64
	Requested int64
}

func (e *InsufficientTroopsError) Error() string {
	return fmt.Sprintf("insufficient %s troops: requested %d, available %d", e.Type, e.Requested, e.Available)
}

func (e *UnreachableDestinationError) Error() string {
	return fmt.Sprintf("army destination is unreachable: (%d, %d)", e.X, e.Y)
}

func (e *ArmyNotFoundError) Error() string {
	return fmt.Sprintf("army not found: %s", e.ArmyID)
}
