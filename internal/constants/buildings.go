package constants

const (
	BUILDING_TYPE_CITY_CENTER = "city_center"
	BUILDING_TYPE_TOWN_CENTER = "town_center"
	BUILDING_TYPE_BARRACKS    = "barracks"
	BUILDING_TYPE_HOUSE       = "house"
	BUILDING_TYPE_FARM        = "farm"
	BUILDING_TYPE_MINE        = "mine"
)

var buildingProduction = map[string][]int64{
	BUILDING_TYPE_CITY_CENTER: {100, 200, 300, 400, 500, 600, 700, 800, 900, 1000},
	BUILDING_TYPE_TOWN_CENTER: {100, 200, 300, 400, 500, 600, 700, 800, 900, 1000},
	BUILDING_TYPE_BARRACKS:    {50, 100, 150, 200, 250, 300, 350, 400, 450, 500},
	BUILDING_TYPE_FARM:        {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	BUILDING_TYPE_MINE:        {30, 60, 90, 120, 150, 180, 210, 240, 270, 300},
}

var buildingPopulation = map[string][]float64{
	BUILDING_TYPE_CITY_CENTER: {1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000},
	BUILDING_TYPE_TOWN_CENTER: {1000, 200, 300, 400, 500, 600, 700, 800, 900, 10000},
	BUILDING_TYPE_HOUSE:       {250, 500, 750, 1000, 1250, 1500, 1750, 2000, 2250, 2500},
}

var buildingCosts = map[string][]int64{
	BUILDING_TYPE_CITY_CENTER: {1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000},
	BUILDING_TYPE_TOWN_CENTER: {1000, 200, 300, 400, 500, 600, 700, 800, 900, 10000},
	BUILDING_TYPE_BARRACKS:    {500, 1000, 1500, 2000, 2500, 3000, 3500, 4000, 4500, 5000},
	BUILDING_TYPE_HOUSE:       {200, 400, 600, 800, 1000, 1200, 1400, 1600, 1800, 2000},
	BUILDING_TYPE_FARM:        {100, 200, 300, 400, 500, 600, 700, 800, 900, 1000},
	BUILDING_TYPE_MINE:        {300, 600, 900, 1200, 1500, 1800, 2100, 2400, 2700, 3000},
}

// in seconds
var buildingConstructionTime = map[string][]int64{
	BUILDING_TYPE_CITY_CENTER: {0, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	BUILDING_TYPE_TOWN_CENTER: {0, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	BUILDING_TYPE_BARRACKS:    {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	BUILDING_TYPE_HOUSE:       {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
	BUILDING_TYPE_FARM:        {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
	BUILDING_TYPE_MINE:        {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
}

func GetBuildingProduction(buildingType string, level int) int64 {
	return buildingProduction[buildingType][level-1]
}

func GetBuildingPopulation(buildingType string, level int) float64 {
	return buildingPopulation[buildingType][level-1]
}

func GetBuildingCost(buildingType string, level int) int64 {
	return buildingCosts[buildingType][level-1]
}

func GetBuildingConstructionTime(buildingType string, level int) int64 {
	return buildingConstructionTime[buildingType][level-1]
}
