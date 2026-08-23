package constants

import (
	"math"

	"cityio/internal/domain"
)

const (
	// CoreCivilianPercent is the housing share protected from recruitment.
	CoreCivilianPercent = 55

	// Militia is a non-mobile settlement reserve. Players start at 10%; neutral
	// towns commit the full non-core population share so they have no recruitable
	// surplus and cannot be captured undefended.
	DefaultMilitiaPercent = 10
	NeutralMilitiaPercent = 45
	MinMilitiaPercent     = 5
	MaxMilitiaPercent     = 100 - CoreCivilianPercent

	// Tax is configured as a whole-number percentage. At 100%, each taxable
	// resident yields TaxGoldPerPopPerHour gold/hour and positive population
	// growth is fully suppressed.
	DefaultTaxRatePercent       = 10
	NeutralTaxRatePercent       = 0
	MaxTaxRatePercent           = 100
	TaxGoldPerPopPerHour  int64 = 16
)

func CorePopulation(city domain.City) float64 {
	return city.PopulationCap * float64(CoreCivilianPercent) / 100
}

func MilitiaTarget(city domain.City) float64 {
	return city.PopulationCap * float64(city.MilitiaPercent) / 100
}

func RecruitablePopulation(city domain.City) int64 {
	available := city.Population - CorePopulation(city) - MilitiaTarget(city)
	return max(int64(math.Floor(available)), 0)
}

func TaxablePopulation(city domain.City) float64 {
	return max(city.Population-city.MilitiaPopulation, 0)
}

func TaxIncomePerHour(city domain.City) int64 {
	return int64(math.Round(
		TaxablePopulation(city) * float64(TaxGoldPerPopPerHour) * float64(city.TaxRatePercent) / 100,
	))
}

func PositiveGrowthMultiplier(taxRatePercent int) float64 {
	return max(0, 1-float64(taxRatePercent)/100)
}
