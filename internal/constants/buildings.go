package constants

import "cityio/internal/domain"

const MAX_BUILDING_LEVEL = 10

// BuildingProductionEntry pairs a resource name with per-level amounts. Amounts
// are stored as integer values per SecondsPerDay (i.e. per day).
type BuildingProductionEntry struct {
	Resource string
	Amounts  []int64
}

// All production values are per day. Chosen so per-tick math
// (amount * tickSeconds / SecondsPerDay) is exact integer division for the
// current 3s tick: each value is a multiple of 28800 = 86400/3.
var buildingProduction = map[domain.BuildingType][]BuildingProductionEntry{
	domain.BuildingTypeCityCenter: {{"gold", []int64{144000, 288000, 432000, 576000, 720000, 864000, 1008000, 1152000, 1296000, 1440000}}},
	domain.BuildingTypeTownCenter: {{"gold", []int64{86400, 172800, 259200, 345600, 432000, 518400, 604800, 691200, 777600, 864000}}},
	domain.BuildingTypeFarm:       {{"food", []int64{288000, 576000, 864000, 1152000, 1440000, 1728000, 2016000, 2304000, 2592000, 2880000}}},
	domain.BuildingTypeMine:       {{"gold", []int64{288000, 576000, 864000, 1152000, 1440000, 1728000, 2016000, 2304000, 2592000, 2880000}}},
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

// GetBuildingProduction returns the per-day production rate for the given
// resource at the given level. Returns 0 if the building does not produce that
// resource.
func GetBuildingProduction(buildingType domain.BuildingType, level int, resource string) int64 {
	for _, entry := range buildingProduction[buildingType] {
		if entry.Resource == resource {
			return entry.Amounts[level-1]
		}
	}
	return 0
}

// PerTickAmount converts a per-day rate to the amount emitted in one tick of
// tickSeconds duration. Exact integer division for rates that are multiples of
// SecondsPerDay / tickSeconds.
func PerTickAmount(perDay int64, tickSeconds int) int64 {
	return perDay * int64(tickSeconds) / SecondsPerDay
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
