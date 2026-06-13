// Package constants contains global constants used throughout the game server.
package constants

const (
	MapSize  = 128 // generate a map of size MapSize x MapSize
	CitySize = 5

	PopulationGrowthRate = 0.001

	InitialTownPopulation = 100

	InitialPlayerCityPopulation = 250
	InitialPlayerGold           = 100000
	InitialPlayerFood           = 100000

	TroopMovementBackupFrequency = 5 // number of tile movements before state saved to db

	// in seconds
	DBBackupFrequency           = 2  // frequency of database flushing buffer queue and writing to database
	UserBackupFrequency         = 10 // frequency of user state being sent to update queue
	CityBackupFrequency         = 10 // frequency of population growth event and city state being sent to update queue
	BuildingProductionFrequency = 3  // frequency of building production

	ActorTimeoutDuration = 2 // timeout on actor response await

	TroopTrainingDuration = 5
	TroopMovementDuration = 1 // time it takes to cross 1 tile
)
