package constants

import "cityio/internal/domain"

const (
	MAX_BUILDING_LEVEL       = 10
	StructurePlacementRadius = 10
)

// BuildingProductionEntry pairs a resource name with per-level amounts. Amounts
// are stored as integer values per SecondsPerHour (i.e. per hour).
type BuildingProductionEntry struct {
	Resource string
	Amounts  []int64
}

// All production values are per hour. Building actors carry fractional
// per-tick output so modifiers preserve these long-run rates exactly.
var buildingProduction = map[domain.BuildingType][]BuildingProductionEntry{
	domain.BuildingTypeCityCenter: {{"gold", []int64{3600, 7200, 10800, 14400, 18000, 21600, 25200, 28800, 32400, 36000}}},
	domain.BuildingTypeTownCenter: {{"gold", []int64{3600, 7200, 10800, 14400, 18000, 21600, 25200, 28800, 32400, 36000}}},
	domain.BuildingTypeFarm:       {{"food", []int64{12000, 24000, 36000, 48000, 60000, 72000, 84000, 96000, 108000, 120000}}},
	domain.BuildingTypeMine:       {{"gold", []int64{7200, 14400, 21600, 28800, 36000, 43200, 50400, 57600, 64800, 72000}}},
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
	domain.BuildingTypeWatchtower: {200, 400, 600, 800, 1000, 1200, 1400, 1600, 1800, 2000},
	domain.BuildingTypeFort:       {500, 1000, 1500, 2000, 2500, 3000, 3500, 4000, 4500, 5000},
}

// in seconds
var buildingConstructionTime = map[domain.BuildingType][]int64{
	domain.BuildingTypeCityCenter: {0, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	domain.BuildingTypeTownCenter: {0, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	domain.BuildingTypeBarracks:   {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
	domain.BuildingTypeHouse:      {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
	domain.BuildingTypeFarm:       {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
	domain.BuildingTypeMine:       {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
	domain.BuildingTypeWatchtower: {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
	domain.BuildingTypeFort:       {10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
}

var buildingVisionRadius = map[domain.BuildingType][]int{
	domain.BuildingTypeWatchtower: {3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
}

var buildingDefenseBonusPercent = map[domain.BuildingType][]int{
	domain.BuildingTypeCityCenter: {15, 20, 25, 30, 35, 40, 45, 50, 55, 60},
	domain.BuildingTypeTownCenter: {10, 15, 20, 25, 30, 35, 40, 45, 50, 55},
	domain.BuildingTypeFort:       {5, 10, 15, 20, 25, 30, 35, 40, 45, 50},
}

// GetBuildingProduction returns the per-hour production rate for the given
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
		domain.BuildingTypeWatchtower,
		domain.BuildingTypeFort,
	}
}

func IsStandaloneStructure(t domain.BuildingType) bool {
	return t == domain.BuildingTypeWatchtower || t == domain.BuildingTypeFort
}

func GetBuildingVisionRadius(t domain.BuildingType, level int) int {
	levels := buildingVisionRadius[t]
	if level <= 0 || level > len(levels) {
		return 0
	}
	return levels[level-1]
}

func GetBuildingDefenseBonusPercent(t domain.BuildingType, level int) int {
	levels := buildingDefenseBonusPercent[t]
	if level <= 0 || level > len(levels) {
		return 0
	}
	return levels[level-1]
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
