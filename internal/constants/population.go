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
	// resident yields TaxGoldPerPopPerHour gold/hour. The growth penalty is
	// deliberately stronger than the tax rate: maximum tax turns normal growth
	// into population decline instead of merely stopping it.
	DefaultTaxRatePercent            = 10
	NeutralTaxRatePercent            = 0
	MaxTaxRatePercent                = 100
	MaxTaxGrowthPenaltyPercent       = 150
	TaxGoldPerPopPerHour       int64 = 16
)

func CorePopulation(city domain.City) float64 {
	basis := city.PopulationBasis
	if basis <= 0 {
		basis = city.Population
	}
	return basis * float64(CoreCivilianPercent) / 100
}

func MilitiaTarget(city domain.City) float64 {
	return max(city.MilitiaTarget, 0)
}

func MilitiaTargetForPercent(city domain.City, percent int) float64 {
	return math.Round(city.PopulationCap * float64(percent) / 100)
}

func MinMilitiaTarget(city domain.City) float64 {
	return math.Ceil(city.PopulationCap * float64(MinMilitiaPercent) / 100)
}

func MaxMilitiaTarget(city domain.City) float64 {
	return math.Floor(city.PopulationCap * float64(MaxMilitiaPercent) / 100)
}

func MilitiaPercent(city domain.City) float64 {
	if city.PopulationCap <= 0 {
		return 0
	}
	return city.MilitiaTarget / city.PopulationCap * 100
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

func TaxGrowthMultiplier(taxRatePercent int) float64 {
	penalty := float64(taxRatePercent) * float64(MaxTaxGrowthPenaltyPercent) / float64(MaxTaxRatePercent) / 100
	return 1 - penalty
}
