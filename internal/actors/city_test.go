package actors

import (
	"math"
	"testing"

	"cityio/internal/constants"
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
