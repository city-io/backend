package constants

import (
	"testing"

	"cityio/internal/domain"
)

func TestPopulationPolicyBuckets(t *testing.T) {
	city := domain.City{
		Population:         250,
		PopulationCap:      250,
		GarrisonPopulation: 25,
		GarrisonPercent:    10,
		TaxRatePercent:     10,
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

func TestRecruitablePopulationProtectsTargetGarrisonAfterLosses(t *testing.T) {
	city := domain.City{
		Population:         200,
		PopulationCap:      250,
		GarrisonPopulation: 5,
		GarrisonPercent:    10,
	}
	if got := RecruitablePopulation(city); got != 37 {
		t.Fatalf("recruitable population = %d, want 37", got)
	}
}
