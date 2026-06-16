package constants

import "cityio/internal/domain"

const MAX_BUILDING_LEVEL = 10

// BuildingProductionEntry pairs a resource name with per-level amounts.
type BuildingProductionEntry struct {
	Resource string
	Amounts  []int64
}

var buildingProduction = map[domain.BuildingType][]BuildingProductionEntry{
	domain.BuildingTypeCityCenter: {{"gold", []int64{5, 10, 15, 20, 25, 30, 35, 40, 45, 50}}},
	domain.BuildingTypeTownCenter: {{"gold", []int64{3, 6, 9, 12, 15, 18, 21, 24, 27, 30}}},
	domain.BuildingTypeFarm:       {{"food", []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}},
	domain.BuildingTypeMine:       {{"gold", []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}}},
}

var buildingPopulation = map[domain.BuildingType][]float64{
	domain.BuildingTypeCityCenter: {250, 350, 450, 550, 650, 750, 850, 950, 1050, 1150},
	domain.BuildingTypeTownCenter: {50, 100, 150, 200, 250, 300, 350, 400, 450, 500},
	domain.BuildingTypeHouse:      {50, 100, 150, 200, 250, 300, 350, 400, 450, 500},
}

var buildingCosts = map[domain.BuildingType][]int64{
	domain.BuildingTypeCityCenter: {1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000},
	domain.BuildingTypeTownCenter: {500, 1000, 1500, 2000, 2500, 3000, 3500, 4000, 4500, 5000},
	domain.BuildingTypeBarracks:   {500, 1000, 1500, 2000, 2500, 3000, 3500, 4000, 4500, 5000},
	domain.BuildingTypeHouse:      {200, 400, 600, 800, 1000, 1200, 1400, 1600, 1800, 2000},
	domain.BuildingTypeFarm:       {300, 600, 900, 1200, 1500, 1800, 2100, 2400, 2700, 3000},
	domain.BuildingTypeMine:       {300, 600, 900, 1200, 1500, 1800, 2100, 2400, 2700, 3000},
}

// in seconds
var buildingConstructionTime = map[domain.BuildingType][]int64{
	domain.BuildingTypeCityCenter: {0, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	domain.BuildingTypeTownCenter: {0, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	domain.BuildingTypeBarracks:   {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	domain.BuildingTypeHouse:      {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
	domain.BuildingTypeFarm:       {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
	domain.BuildingTypeMine:       {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
}

func GetBuildingProduction(buildingType domain.BuildingType, level int) int64 {
	entries := buildingProduction[buildingType]
	if len(entries) == 0 {
		return 0
	}
	return entries[0].Amounts[level-1]
}

func GetBuildingProductionEntries(buildingType domain.BuildingType) []BuildingProductionEntry {
	return buildingProduction[buildingType]
}

func GetBuildingPopulation(buildingType domain.BuildingType, level int) float64 {
	return buildingPopulation[buildingType][level-1]
}

func GetBuildingCost(buildingType domain.BuildingType, level int) int64 {
	return buildingCosts[buildingType][level-1]
}

func GetBuildingConstructionTime(buildingType domain.BuildingType, level int) int64 {
	return buildingConstructionTime[buildingType][level-1]
}

// AllBuildingTypes returns every building type that has a cost table defined.
func AllBuildingTypes() []domain.BuildingType {
	return []domain.BuildingType{
		domain.BuildingTypeCityCenter,
		domain.BuildingTypeTownCenter,
		domain.BuildingTypeBarracks,
		domain.BuildingTypeHouse,
		domain.BuildingTypeFarm,
		domain.BuildingTypeMine,
	}
}

func GetBuildingCosts(buildingType domain.BuildingType) []int64 {
	return buildingCosts[buildingType]
}

func GetBuildingConstructionTimes(buildingType domain.BuildingType) []int64 {
	return buildingConstructionTime[buildingType]
}

func GetBuildingPopulations(buildingType domain.BuildingType) []float64 {
	return buildingPopulation[buildingType]
}
