// Package constants contains global constants used throughout the game server.
package constants

const (
	MapSize  = 75 // generate a map of size MapSize x MapSize
	CitySize = 5

	// SecondsPerDay is the canonical period for all rate values: production,
	// upkeep, etc. are stored, computed, and shipped as int64 amounts per this
	// many seconds.
	SecondsPerDay = 86400

	// PopulationGrowthRate is applied to current population on every city tick.
	// Retuned for a 3s tick so growth feels equivalent to the previous 10s
	// cadence: 0.001 per 10s ≈ 0.0003 per 3s.
	PopulationGrowthRate = 0.0003

	// FoodPerPopPerDay is the per-population food upkeep per day. 250 pop × 1152
	// = 288,000 food/day, exactly one L1 farm's output.
	FoodPerPopPerDay int64 = 1152

	// StarvationDeclineRate scales population loss per tick when a city is
	// starving. Applied as pop *= (1 - rate * shortfall_ratio).
	StarvationDeclineRate = 0.005

	InitialPlayerCityPopulation = 250

	InitialPlayerGold = 2000
	InitialPlayerFood = 1000

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
