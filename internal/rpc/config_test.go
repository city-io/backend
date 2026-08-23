package rpc

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"cityio/internal/constants"
	servicev1 "cityio/internal/gen/cityio/service/v1"
)

func TestGameConfigExposesPopulationPolicy(t *testing.T) {
	response, err := (&configHandler{}).GetGameConfig(
		context.Background(),
		connect.NewRequest(&servicev1.GetGameConfigRequest{}),
	)
	if err != nil {
		t.Fatalf("get game config: %v", err)
	}
	policy := response.Msg.GetPopulationPolicy()
	if policy.GetCoreCivilianPercent() != constants.CoreCivilianPercent ||
		policy.GetDefaultGarrisonPercent() != constants.DefaultGarrisonPercent ||
		policy.GetMinGarrisonPercent() != constants.MinGarrisonPercent ||
		policy.GetMaxGarrisonPercent() != constants.MaxGarrisonPercent ||
		policy.GetMaxMilitaryPercent() != 100-constants.CoreCivilianPercent ||
		policy.GetMaxTaxRatePercent() != constants.MaxTaxRatePercent {
		t.Fatalf("population policy = %+v", policy)
	}
	if policy.GetTaxGoldPerPopulation().GetValue() != constants.TaxGoldPerPopPerHour || policy.GetTaxGoldPerPopulation().GetScale() != constants.SecondsPerHour {
		t.Fatalf("tax rate = %+v", policy.GetTaxGoldPerPopulation())
	}
}
