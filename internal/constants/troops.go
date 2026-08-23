package constants

import "cityio/internal/domain"

// MilitaryPopulationFraction is the share of a city's population that can be
// reserved as standing army. The rest is an untrainable civilian core.
const MilitaryPopulationFraction = 0.35

// TroopStat holds the tier-1 stat profile for a troop type. Gold is the
// per-troop training cost; TrainTime is per troop in seconds; FoodUpkeep is per hour;
// PopCost is the population reserved per troop. MovementTicks is the base
// number of 250ms updates needed to enter normal terrain. Attack/Defense/HP are
// stored now but unused until combat.
type TroopStat struct {
	Gold          int64
	TrainTime     int64
	FoodUpkeep    int64
	PopCost       int64
	MovementTicks int
	Attack        int64
	Defense       int64
	HP            int64
}

var troopStats = map[domain.TroopType]TroopStat{
	domain.TroopTypeSoldier:   {Gold: 50, TrainTime: 5, FoodUpkeep: 60, PopCost: 1, MovementTicks: 4, Attack: 10, Defense: 10, HP: 100},
	domain.TroopTypeArcher:    {Gold: 75, TrainTime: 7, FoodUpkeep: 60, PopCost: 1, MovementTicks: 4, Attack: 15, Defense: 5, HP: 70},
	domain.TroopTypeCavalry:   {Gold: 150, TrainTime: 10, FoodUpkeep: 180, PopCost: 1, MovementTicks: 2, Attack: 20, Defense: 12, HP: 120},
	domain.TroopTypeArtillery: {Gold: 300, TrainTime: 15, FoodUpkeep: 120, PopCost: 3, MovementTicks: 6, Attack: 40, Defense: 3, HP: 60},
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

// GetTroopMovementTicks returns the base movement updates for a troop type.
func GetTroopMovementTicks(t domain.TroopType) int {
	return troopStats[t].MovementTicks
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
