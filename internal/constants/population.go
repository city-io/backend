package constants

import (
	"math"

	"cityio/internal/domain"
)

func CorePopulation(city domain.City) float64 {
	return city.PopulationCap * float64(CoreCivilianPercent) / 100
}

func GarrisonTarget(city domain.City) float64 {
	return city.PopulationCap * float64(city.GarrisonPercent) / 100
}

func RecruitablePopulation(city domain.City) int64 {
	available := city.Population - CorePopulation(city) - GarrisonTarget(city)
	return max(int64(math.Floor(available)), 0)
}

func TaxablePopulation(city domain.City) float64 {
	return max(city.Population-city.GarrisonPopulation, 0)
}

func TaxIncomePerHour(city domain.City) int64 {
	return int64(math.Round(
		TaxablePopulation(city) * float64(TaxGoldPerPopPerHour) * float64(city.TaxRatePercent) / 100,
	))
}

func PositiveGrowthMultiplier(taxRatePercent int) float64 {
	return max(0, 1-float64(taxRatePercent)/100)
}
