package actors

import (
	"math"
	"testing"

	"cityio/internal/constants"
	"cityio/internal/domain"
)

func TestCityFoodBalanceStaysStarvingAcrossRoundedDemandTicks(t *testing.T) {
	var remainder int64
	sawTen, sawEleven := false, false
	for range 20 {
		demand := rateAmountForTick(12288, constants.CityTickInterval, &remainder)
		sawTen = sawTen || demand == 10
		sawEleven = sawEleven || demand == 11

		balance := calculateCityFoodBalance(12000, 12288)
		if !balance.starving {
			t.Fatalf("city became fed on a %d-food demand tick", demand)
		}
		if math.Abs(balance.deficitRatio-288.0/12288.0) > 1e-12 {
			t.Fatalf("deficit ratio = %f", balance.deficitRatio)
		}
	}
	if !sawTen || !sawEleven {
		t.Fatalf("demand ticks did not exercise rounding boundary: saw10=%v saw11=%v", sawTen, sawEleven)
	}
}

func TestCityFoodBalanceTreatsExactHourlyBalanceAsFed(t *testing.T) {
	balance := calculateCityFoodBalance(12000, 12000)
	if balance.starving || balance.deficitRatio != 0 || balance.surplusRatio != 0 {
		t.Fatalf("balanced city state = %+v", balance)
	}
}

func TestCityFoodProductionTotalsBuildingRates(t *testing.T) {
	state := cityActor{foodProduction: map[string]int64{"farm-1": 12000, "farm-2": 24000}}
	if got := state.foodProductionTotal(); got != 36000 {
		t.Fatalf("food production = %d, want 36000", got)
	}
}

func TestCityGrowthRefillsGarrisonBeforeTaxablePopulation(t *testing.T) {
	state := cityActor{City: domain.City{
		Population:         200,
		PopulationCap:      250,
		GarrisonPopulation: 5,
		GarrisonPercent:    10,
	}}
	state.growPopulation(false, 0, 0)
	delta := state.City.Population - 200
	if delta <= 0 {
		t.Fatal("population did not grow")
	}
	if math.Abs(state.City.GarrisonPopulation-(5+delta)) > 1e-12 {
		t.Fatalf("garrison = %f, want %f", state.City.GarrisonPopulation, 5+delta)
	}
}

func TestMaximumTaxStopsPositiveGrowth(t *testing.T) {
	state := cityActor{City: domain.City{
		Population:      200,
		PopulationCap:   250,
		TaxRatePercent:  constants.MaxTaxRatePercent,
		GarrisonPercent: 10,
	}}
	state.growPopulation(false, 0, 1)
	if state.City.Population != 200 || state.City.PopulationGrowthRate != 0 {
		t.Fatalf("population changed under maximum tax: %+v", state.City)
	}
}

func TestRecruitmentTransfersResidentsOutOfCity(t *testing.T) {
	state := cityActor{City: domain.City{
		Population: 250, PopulationCap: 250, GarrisonPopulation: 25,
		GarrisonPercent: 10, TaxRatePercent: 10,
	}}
	if err := state.recruitPopulation(45); err != nil {
		t.Fatalf("recruitment failed: %v", err)
	}
	if state.City.Population != 205 || constants.TaxablePopulation(state.City) != 180 {
		t.Fatalf("city after recruitment = %+v", state.City)
	}
	if got := constants.RecruitablePopulation(state.City); got != 42 {
		t.Fatalf("recruitable population = %d, want 42", got)
	}
}

func TestRecruitmentCannotCrossCoreAndGarrisonFloor(t *testing.T) {
	state := cityActor{City: domain.City{
		Population: 163, PopulationCap: 250, GarrisonPopulation: 25,
		GarrisonPercent: 10,
	}}
	err := state.recruitPopulation(1)
	if err == nil || err.Available != 0 {
		t.Fatalf("recruitment error = %+v, want zero available", err)
	}
}

func TestGarrisonCasualtiesReduceSettlementPopulation(t *testing.T) {
	state := cityActor{City: domain.City{Population: 250, GarrisonPopulation: 25}}
	if !state.applyGarrisonCasualties(5) {
		t.Fatal("garrison unexpectedly destroyed")
	}
	if state.City.Population != 245 || state.City.GarrisonPopulation != 20 {
		t.Fatalf("city after casualties = %+v", state.City)
	}
}
