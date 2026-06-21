// Package constants contains global constants used throughout the game server.
package constants

const (
	MapSize  = 75 // generate a map of size MapSize x MapSize
	CitySize = 5

	// SecondsPerHour is the canonical period for all rate values: production,
	// upkeep, etc. are stored, computed, and shipped as int64 amounts per this
	// many seconds. Per-hour reads more naturally for players than per-day,
	// and the same clean-tick property holds (3600 % 3 == 0).
	SecondsPerHour = 3600

	// PopulationGrowthRate is the base logistic growth rate per city tick, in
	// the absence of any food surplus bonus. Slow on purpose so a food
	// surplus is meaningfully felt — a city that only just covers its upkeep
	// grows at this rate, and stacking extra farms speeds it up via
	// SurplusGrowthBonus.
	PopulationGrowthRate = 0.0002

	// SurplusGrowthBonus is the maximum additional growth multiplier from a
	// food surplus. The bonus saturates at 100% surplus (production = 2×
	// demand): below that it scales linearly; above it stays capped. At full
	// saturation growth runs at (1 + SurplusGrowthBonus)× the base rate.
	SurplusGrowthBonus = 1.0

	// FoodPerPopPerHour is the per-population food upkeep per hour. 250 pop ×
	// 48 = 12,000 food/hour, exactly one L1 farm's output.
	FoodPerPopPerHour int64 = 48

	// StarvationDeclineRate scales population loss per tick when a city's
	// own production doesn't cover its demand. Applied as
	// pop *= (1 - rate * deficitRatio). Pool coverage no longer prevents the
	// decline — a city has to be locally self-sufficient to hold or grow.
	StarvationDeclineRate = 0.005

	InitialPlayerCityPopulation = 250

	InitialPlayerGold = 5000
	InitialPlayerFood = 5000

	TroopMovementBackupFrequency = 5 // number of tile movements before state saved to db

	// in seconds
	DBBackupFrequency    = 2  // frequency of database flushing buffer queue and writing to database
	UserBackupFrequency  = 10 // frequency of user state being sent to update queue
	CityTickInterval     = 3  // cadence of the city actor: food loop, population growth, stream push, backup enqueue
	BuildingTickInterval = 3  // cadence of building actors: resource production, construction checks, tile reaffirm

	ActorTimeoutDuration = 2 // timeout on actor response await

	TroopTrainingDuration = 5
	TroopMovementDuration = 1 // time it takes to cross 1 tile

	VisionRadius = 3 // Chebyshev distance beyond owned city edges that a player can see
)

type TownConfig struct {
	CenterLevel int
	HouseCount  int
}

// TC L1=50, L2=100, L3=150; House L1=50
var TownSizeConfig = map[int]TownConfig{
	2: {CenterLevel: 1, HouseCount: 1}, // 50 + 50 = 100
	3: {CenterLevel: 1, HouseCount: 2}, // 50 + 100 = 150
	4: {CenterLevel: 2, HouseCount: 2}, // 100 + 100 = 200
	5: {CenterLevel: 3, HouseCount: 2}, // 150 + 100 = 250
}
