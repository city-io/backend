package constants

import "cityio/internal/models"

const (
	BuildingTypeCityCenter models.BuildingType = "city_center"
	BuildingTypeTownCenter models.BuildingType = "town_center"
	BuildingTypeBarracks   models.BuildingType = "barracks"
	BuildingTypeHouse      models.BuildingType = "house"
	BuildingTypeFarm       models.BuildingType = "farm"
	BuildingTypeMine       models.BuildingType = "mine"
)

const MAX_BUILDING_LEVEL = 10

var buildingProduction = map[models.BuildingType][]int64{
	BuildingTypeCityCenter: {100, 200, 300, 400, 500, 600, 700, 800, 900, 1000},
	BuildingTypeTownCenter: {100, 200, 300, 400, 500, 600, 700, 800, 900, 1000},
	BuildingTypeBarracks:   {50, 100, 150, 200, 250, 300, 350, 400, 450, 500},
	BuildingTypeFarm:       {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	BuildingTypeMine:       {30, 60, 90, 120, 150, 180, 210, 240, 270, 300},
}

var buildingPopulation = map[models.BuildingType][]float64{
	BuildingTypeCityCenter: {1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000},
	BuildingTypeTownCenter: {1000, 200, 300, 400, 500, 600, 700, 800, 900, 10000},
	BuildingTypeHouse:      {250, 500, 750, 1000, 1250, 1500, 1750, 2000, 2250, 2500},
}

var buildingCosts = map[models.BuildingType][]int64{
	BuildingTypeCityCenter: {1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000},
	BuildingTypeTownCenter: {1000, 200, 300, 400, 500, 600, 700, 800, 900, 10000},
	BuildingTypeBarracks:   {500, 1000, 1500, 2000, 2500, 3000, 3500, 4000, 4500, 5000},
	BuildingTypeHouse:      {200, 400, 600, 800, 1000, 1200, 1400, 1600, 1800, 2000},
	BuildingTypeFarm:       {100, 200, 300, 400, 500, 600, 700, 800, 900, 1000},
	BuildingTypeMine:       {300, 600, 900, 1200, 1500, 1800, 2100, 2400, 2700, 3000},
}

// in seconds
var buildingConstructionTime = map[models.BuildingType][]int64{
	BuildingTypeCityCenter: {0, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	BuildingTypeTownCenter: {0, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	BuildingTypeBarracks:   {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	BuildingTypeHouse:      {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
	BuildingTypeFarm:       {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
	BuildingTypeMine:       {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
}

func GetBuildingProduction(buildingType models.BuildingType, level int) int64 {
	return buildingProduction[buildingType][level-1]
}

func GetBuildingPopulation(buildingType models.BuildingType, level int) float64 {
	return buildingPopulation[buildingType][level-1]
}

func GetBuildingCost(buildingType models.BuildingType, level int) int64 {
	return buildingCosts[buildingType][level-1]
}

func GetBuildingConstructionTime(buildingType models.BuildingType, level int) int64 {
	return buildingConstructionTime[buildingType][level-1]
}
