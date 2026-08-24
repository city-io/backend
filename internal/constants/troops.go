package constants

import (
	"time"

	"cityio/internal/domain"
)

const (
	BattleTickInterval        = 5 * time.Second
	SettlementCaptureDuration = 30 * time.Second
	// BattleCasualtyRate scales military losses calculated from incoming power.
	// Keeping it below one lets formations exchange several rounds before one
	// side collapses.
	BattleCasualtyRate = 0.50
	// SiegeCivilianCasualtyRate converts incoming military force into a much
	// smaller amount of collateral population loss, accumulated across rounds.
	SiegeCivilianCasualtyRate = 0.10
)

// TroopStat holds the tier-1 stat profile for a troop type. Gold is the
// per-troop training cost; TrainTime is per troop in seconds; FoodUpkeep is per hour;
// PopCost is the population reserved per troop. MovementDuration is the base
// time needed to enter normal terrain. Attack/Defense/HP drive battle losses.
type TroopStat struct {
	Gold             int64
	TrainTime        int64
	FoodUpkeep       int64
	PopCost          int64
	MovementDuration time.Duration
	Attack           int64
	Defense          int64
	HP               int64
}

var troopStats = map[domain.TroopType]TroopStat{
	domain.TroopTypeSoldier:   {Gold: 50, TrainTime: 5, FoodUpkeep: 60, PopCost: 1, MovementDuration: 1650 * time.Millisecond, Attack: 10, Defense: 10, HP: 100},
	domain.TroopTypeArcher:    {Gold: 75, TrainTime: 7, FoodUpkeep: 60, PopCost: 1, MovementDuration: 1650 * time.Millisecond, Attack: 15, Defense: 5, HP: 70},
	domain.TroopTypeCavalry:   {Gold: 150, TrainTime: 10, FoodUpkeep: 180, PopCost: 1, MovementDuration: 825 * time.Millisecond, Attack: 20, Defense: 12, HP: 120},
	domain.TroopTypeArtillery: {Gold: 300, TrainTime: 15, FoodUpkeep: 120, PopCost: 3, MovementDuration: 2475 * time.Millisecond, Attack: 40, Defense: 3, HP: 60},
}

// GetTroopStat returns the full stat profile for a troop type.
func GetTroopStat(t domain.TroopType) TroopStat {
	return troopStats[t]
}

// GetTroopGoldCost returns the per-troop training gold cost.
func GetTroopGoldCost(t domain.TroopType) int64 {
	return troopStats[t].Gold
}

// GetTroopTrainTime returns the per-troop training duration in seconds.
func GetTroopTrainTime(t domain.TroopType) int64 {
	return troopStats[t].TrainTime
}

// GetTroopTrainingDuration returns the total batch duration in seconds.
func GetTroopTrainingDuration(t domain.TroopType, count int64) int64 {
	return GetTroopTrainTime(t) * count
}

// GetTroopFoodUpkeep returns the per-troop food upkeep per hour.
func GetTroopFoodUpkeep(t domain.TroopType) int64 {
	return troopStats[t].FoodUpkeep
}

// GetTroopPopCost returns the population reserved per troop.
func GetTroopPopCost(t domain.TroopType) int64 {
	return troopStats[t].PopCost
}

// GetTroopMovementDuration returns the base normal-terrain movement time.
func GetTroopMovementDuration(t domain.TroopType) time.Duration {
	return troopStats[t].MovementDuration
}

// GetBarracksTrainingCapacity returns how many troops a barracks of the given
// level can hold in a single in-progress training batch (5 × level).
func GetBarracksTrainingCapacity(level int) int64 {
	return int64(5 * level)
}

// AllTroopTypes returns every defined troop type.
func AllTroopTypes() []domain.TroopType {
	return []domain.TroopType{
		domain.TroopTypeSoldier,
		domain.TroopTypeArcher,
		domain.TroopTypeCavalry,
		domain.TroopTypeArtillery,
	}
}

// IsValidTroopType reports whether t has a defined stat profile.
func IsValidTroopType(t domain.TroopType) bool {
	_, ok := troopStats[t]
	return ok
}
