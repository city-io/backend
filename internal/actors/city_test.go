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

func TestCityGrowthRefillsMilitiaBeforeTaxablePopulation(t *testing.T) {
	state := cityActor{City: domain.City{
		Population:        200,
		PopulationCap:     250,
		PopulationBasis:   200,
		MilitiaPopulation: 5,
		MilitiaTarget:     25,
	}}
	state.growPopulation(false, 0, 0)
	delta := state.City.Population - 200
	if delta <= 0 {
		t.Fatal("population did not grow")
	}
	if math.Abs(state.City.MilitiaPopulation-(5+delta)) > 1e-12 {
		t.Fatalf("militia = %f, want %f", state.City.MilitiaPopulation, 5+delta)
	}
}

func TestMaximumTaxReversesPositiveGrowth(t *testing.T) {
	state := cityActor{City: domain.City{
		Population:      200,
		PopulationCap:   250,
		PopulationBasis: 200,
		TaxRatePercent:  constants.MaxTaxRatePercent,
		MilitiaTarget:   25,
	}}
	state.growPopulation(false, 0, 1)
	if state.City.Population >= 200 || state.City.PopulationGrowthRate >= 0 {
		t.Fatalf("population did not decline under maximum tax: %+v", state.City)
	}
	if state.City.PopulationGrowthBeforeTaxRate <= 0 {
		t.Fatalf("untaxed growth baseline = %d, want positive", state.City.PopulationGrowthBeforeTaxRate)
	}
	if got, want := state.City.PopulationGrowthRate, -state.City.PopulationGrowthBeforeTaxRate/2; math.Abs(float64(got-want)) > 1 {
		t.Fatalf("taxed growth = %d, want about %d", got, want)
	}
}

func TestStarvationReducesCoreAfterRecruitablePopulationIsGone(t *testing.T) {
	state := cityActor{City: domain.City{
		Population: 130, PopulationCap: 250, PopulationBasis: 250,
		MilitiaPopulation: 25, MilitiaTarget: 25,
	}}
	coreBefore := constants.CorePopulation(state.City)
	state.growPopulation(true, 1, 0)

	if state.City.MilitiaPopulation != 25 {
		t.Fatalf("militia population = %f, want 25", state.City.MilitiaPopulation)
	}
	if coreAfter := constants.CorePopulation(state.City); coreAfter >= coreBefore {
		t.Fatalf("core population after starvation = %f, want less than %f", coreAfter, coreBefore)
	}
	if got := constants.RecruitablePopulation(state.City); got != 0 {
		t.Fatalf("recruitable population after starvation = %d, want 0", got)
	}
}

func TestRecruitmentTransfersResidentsOutOfCity(t *testing.T) {
	state := cityActor{City: domain.City{
		Population: 250, PopulationCap: 250, PopulationBasis: 250, MilitiaPopulation: 25,
		MilitiaTarget: 25, TaxRatePercent: 10,
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

func TestRecruitmentCannotCrossCoreAndMilitiaFloor(t *testing.T) {
	state := cityActor{City: domain.City{
		Population: 163, PopulationCap: 250, PopulationBasis: 250, MilitiaPopulation: 25,
		MilitiaTarget: 25,
	}}
	err := state.recruitPopulation(1)
	if err == nil || err.Available != 0 {
		t.Fatalf("recruitment error = %+v, want zero available", err)
	}
}

func TestCityPolicyReassignsNonCoreResidentsToMilitia(t *testing.T) {
	state := cityActor{City: domain.City{
		Population: 220, PopulationCap: 250, PopulationBasis: 250, MilitiaPopulation: 25,
		MilitiaTarget: 25, TaxRatePercent: 10,
	}}

	state.updatePolicy(63, 10)
	if state.City.MilitiaPopulation != 63 {
		t.Fatalf("militia after target increase = %f, want 63", state.City.MilitiaPopulation)
	}
	if got := constants.RecruitablePopulation(state.City); got != 19 {
		t.Fatalf("recruitable population after target increase = %d, want 19", got)
	}

	state.updatePolicy(13, 10)
	if state.City.MilitiaPopulation != 13 {
		t.Fatalf("militia after target decrease = %f, want 13", state.City.MilitiaPopulation)
	}
	if got := constants.RecruitablePopulation(state.City); got != 69 {
		t.Fatalf("recruitable population after target decrease = %d, want 69", got)
	}
}

func TestCityPolicyPreservesMilitiaAfterHousingExpansion(t *testing.T) {
	state := cityActor{City: domain.City{
		Population: 249, PopulationCap: 800, PopulationBasis: 250,
		MilitiaPopulation: 25, MilitiaTarget: 25, TaxRatePercent: 10,
	}}

	state.updatePolicy(40, 10)
	if state.City.Population != 249 {
		t.Fatalf("population after policy change = %f, want 249", state.City.Population)
	}
	if state.City.MilitiaPopulation != 40 {
		t.Fatalf("militia after housing expansion policy change = %f, want 40", state.City.MilitiaPopulation)
	}
	if got := constants.RecruitablePopulation(state.City); got != 71 {
		t.Fatalf("recruitable population after housing expansion = %d, want 71", got)
	}
}

func TestTaxPolicyChangeDoesNotRefillMilitiaLosses(t *testing.T) {
	state := cityActor{City: domain.City{
		Population: 220, PopulationCap: 250, PopulationBasis: 250, MilitiaPopulation: 20,
		MilitiaTarget: 63, TaxRatePercent: 10, PopulationGrowthRate: 100, PopulationGrowthBeforeTaxRate: 100,
	}}

	state.updatePolicy(63, constants.MaxTaxRatePercent)
	if state.City.MilitiaPopulation != 20 {
		t.Fatalf("militia after tax-only policy change = %f, want 20", state.City.MilitiaPopulation)
	}
	if state.City.PopulationGrowthRate != -50 {
		t.Fatalf("population growth after tax policy change = %d, want -50", state.City.PopulationGrowthRate)
	}
}

func TestMilitiaCasualtiesReduceSettlementPopulation(t *testing.T) {
	state := cityActor{City: domain.City{Population: 250, MilitiaPopulation: 25}}
	if !state.applyMilitiaCasualties(5) {
		t.Fatal("militia unexpectedly destroyed")
	}
	if state.City.Population != 245 || state.City.MilitiaPopulation != 20 {
		t.Fatalf("city after casualties = %+v", state.City)
	}
}

func TestCivilianCasualtiesLeaveMilitiaUntouched(t *testing.T) {
	state := cityActor{City: domain.City{
		Population: 250, PopulationCap: 250, PopulationBasis: 250,
		MilitiaPopulation: 25, TaxRatePercent: 10,
	}}

	if applied := state.applyCivilianCasualties(7); applied != 7 {
		t.Fatalf("applied civilian casualties = %d, want 7", applied)
	}
	if state.City.Population != 243 || state.City.MilitiaPopulation != 25 {
		t.Fatalf("city after civilian casualties = %+v", state.City)
	}
}
