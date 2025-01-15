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
	BUILDING_TYPE_HOUSE:       {20, 40, 60, 80, 100, 120, 140, 160, 180, 200},
	BUILDING_TYPE_FARM:        {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	BUILDING_TYPE_MINE:        {30, 60, 90, 120, 150, 180, 210, 240, 270, 300},
}

var buildingCosts = map[string][]int64{
	BUILDING_TYPE_CITY_CENTER: {100, 200, 300, 400, 500, 600, 700, 800, 900, 1000},
	BUILDING_TYPE_TOWN_CENTER: {100, 200, 300, 400, 500, 600, 700, 800, 900, 1000},
	BUILDING_TYPE_BARRACKS:    {50, 100, 150, 200, 250, 300, 350, 400, 450, 500},
	BUILDING_TYPE_HOUSE:       {20, 40, 60, 80, 100, 120, 140, 160, 180, 200},
	BUILDING_TYPE_FARM:        {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	BUILDING_TYPE_MINE:        {30, 60, 90, 120, 150, 180, 210, 240, 270, 300},
}

func GetBuildingProduction(buildingType string, level int) int64 {
	return buildingProduction[buildingType][level]
}

func GetBuildingCost(buildingType string, level int) int64 {
	return buildingCosts[buildingType][level]
}
