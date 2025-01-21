package constants

const (
	MAP_SIZE  = 128 // generate a map of size MAP_SIZE x MAP_SIZE
	CITY_SIZE = 5

	POPULATION_GROWTH_RATE = 0.001

	INITIAL_TOWN_POPULATION = 100

	INITIAL_PLAYER_CITY_POPULATION = 250
	INITIAL_PLAYER_GOLD            = 100000
	INITIAL_PLAYER_FOOD            = 100000

	// in seconds
	DB_BACKUP_FREQUENCY           = 2  // frequency of database flushing buffer queue and writing to database
	USER_BACKUP_FREQUENCY         = 10 // frequency of user state being sent to update queue
	CITY_BACKUP_FREQUENCY         = 10 // frequency of population growth event and city state being sent to update queue
	BUILDING_PRODUCTION_FREQUENCY = 3  // frequency of building production

	ACTOR_TIMEOUT_DURATION = 1 // timeout on actor response await

	TROOP_TRAINING_DURATION = 5
	TROOP_MOVEMENT_DURATION = 1 // time it takes to cross 1 tile
)
