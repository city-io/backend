package constants

import "cityio/internal/domain"

// MilitaryPopulationFraction is the share of a city's population that can be
// reserved as standing army. The rest is an untrainable civilian core.
const MilitaryPopulationFraction = 0.35

// TroopStat holds the tier-1 stat profile for a troop type. Gold is the
// per-troop training cost; TrainTime is in seconds; FoodUpkeep is per hour;
// PopCost is the population reserved per troop. Attack/Defense/HP are stored
// now but unused until combat. Speed is reserved for later per-type movement
// differentiation (all types move at TroopMovementDuration for now). Sight is
// how many tiles the troop can see from where it stands — cavalry scout wide,
// artillery is near-blind — so an army's reach depends on what it is made of.
type TroopStat struct {
	Gold       int64
	TrainTime  int64
	FoodUpkeep int64
	PopCost    int64
	Attack     int64
	Defense    int64
	HP         int64
	Speed      int64
	Sight      int
}

var troopStats = map[domain.TroopType]TroopStat{
	domain.TroopTypeSoldier:   {Gold: 50, TrainTime: 20, FoodUpkeep: 60, PopCost: 1, Attack: 10, Defense: 10, HP: 100, Speed: 1, Sight: 2},
	domain.TroopTypeArcher:    {Gold: 75, TrainTime: 30, FoodUpkeep: 60, PopCost: 1, Attack: 15, Defense: 5, HP: 70, Speed: 1, Sight: 3},
	domain.TroopTypeCavalry:   {Gold: 150, TrainTime: 45, FoodUpkeep: 180, PopCost: 1, Attack: 20, Defense: 12, HP: 120, Speed: 1, Sight: 4},
	domain.TroopTypeArtillery: {Gold: 300, TrainTime: 60, FoodUpkeep: 120, PopCost: 3, Attack: 40, Defense: 3, HP: 60, Speed: 1, Sight: 1},
}

// CityVisionRadius is how far a city sees beyond its own footprint. Kept
// separate from troop sight so a settlement is not blind while it has no
// garrison. Generous on purpose: until units can carry vision out onto the
// map, this disk is the whole of what a new player sees, and a tight one makes
// the world feel like a corridor.
const CityVisionRadius = 5

// ArmySight is how far an army can see: the widest sight among the troop types
// it contains, so adding a single cavalry scout extends the whole army's reach.
// A stack that somehow holds no troops sees only its own tile.
func ArmySight(troops map[domain.TroopType]int64) int {
	best := 0
	for t, n := range troops {
		if n <= 0 {
			continue
		}
		if s := troopStats[t].Sight; s > best {
			best = s
		}
	}
	return best
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

// GetTroopFoodUpkeep returns the per-troop food upkeep per hour.
func GetTroopFoodUpkeep(t domain.TroopType) int64 {
	return troopStats[t].FoodUpkeep
}

// GetTroopPopCost returns the population reserved per troop.
func GetTroopPopCost(t domain.TroopType) int64 {
	return troopStats[t].PopCost
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
