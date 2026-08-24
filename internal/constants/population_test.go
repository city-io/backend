package constants

import (
	"testing"

	"cityio/internal/domain"
)

func TestPopulationPolicyBuckets(t *testing.T) {
	city := domain.City{
		Population:        250,
		PopulationCap:     250,
		PopulationBasis:   250,
		MilitiaPopulation: 25,
		MilitiaTarget:     25,
		TaxRatePercent:    10,
	}
	if got := CorePopulation(city); got != 137.5 {
		t.Fatalf("core population = %v, want 137.5", got)
	}
	if got := RecruitablePopulation(city); got != 87 {
		t.Fatalf("recruitable population = %d, want 87", got)
	}
	if got := TaxablePopulation(city); got != 225 {
		t.Fatalf("taxable population = %v, want 225", got)
	}
	if got := TaxIncomePerHour(city); got != 360 {
		t.Fatalf("tax income = %d, want 360", got)
	}
}

func TestPopulationBucketsRemainCompleteAfterMilitiaLosses(t *testing.T) {
	city := domain.City{
		Population:        200,
		PopulationCap:     250,
		PopulationBasis:   250,
		MilitiaPopulation: 5,
		MilitiaTarget:     25,
	}
	if got := RecruitablePopulation(city); got != 57 {
		t.Fatalf("recruitable population = %d, want 57", got)
	}
	if got := CorePopulation(city) + RecruitablePopulationExact(city); got != TaxablePopulation(city) {
		t.Fatalf("civilian population buckets = %v, want %v", got, TaxablePopulation(city))
	}
}

func TestCorePopulationFallsWhenStarvationExhaustsRecruitableCivilians(t *testing.T) {
	city := domain.City{
		Population:        150,
		PopulationCap:     250,
		PopulationBasis:   250,
		MilitiaPopulation: 25,
		MilitiaTarget:     25,
	}
	if got := CorePopulation(city); got != 125 {
		t.Fatalf("core population = %v, want all 125 surviving civilians", got)
	}
	if got := RecruitablePopulation(city); got != 0 {
		t.Fatalf("recruitable population = %d, want 0", got)
	}
}

func TestNeutralMilitiaUsesEntireNonCorePopulationShare(t *testing.T) {
	if NeutralMilitiaPercent != MaxMilitiaPercent {
		t.Fatalf("neutral militia = %d%%, maximum = %d%%", NeutralMilitiaPercent, MaxMilitiaPercent)
	}
	if CoreCivilianPercent+NeutralMilitiaPercent != 100 {
		t.Fatalf("core plus neutral militia = %d%%, want 100%%", CoreCivilianPercent+NeutralMilitiaPercent)
	}

	town := domain.City{
		Population:        250,
		PopulationCap:     250,
		PopulationBasis:   250,
		MilitiaPopulation: 250 * float64(NeutralMilitiaPercent) / 100,
		MilitiaTarget:     MaxMilitiaTarget(domain.City{PopulationCap: 250}),
	}
	if got := RecruitablePopulation(town); got != 0 {
		t.Fatalf("neutral town recruitable population = %d, want 0", got)
	}
}

func TestTaxGrowthMultiplierCanReverseGrowth(t *testing.T) {
	if got := TaxGrowthMultiplier(0); got != 1 {
		t.Fatalf("zero-tax growth multiplier = %v, want 1", got)
	}
	if got := TaxGrowthMultiplier(MaxTaxRatePercent); got != -0.5 {
		t.Fatalf("maximum-tax growth multiplier = %v, want -0.5", got)
	}
}
