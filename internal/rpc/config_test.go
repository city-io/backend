package rpc

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"cityio/internal/constants"
	"cityio/internal/domain"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/mapping"
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
		policy.GetDefaultMilitiaPercent() != constants.DefaultMilitiaPercent ||
		policy.GetMinMilitiaPercent() != constants.MinMilitiaPercent ||
		policy.GetMaxMilitiaPercent() != constants.MaxMilitiaPercent ||
		policy.GetMaxMilitaryPercent() != 100-constants.CoreCivilianPercent ||
		policy.GetMaxTaxRatePercent() != constants.MaxTaxRatePercent ||
		policy.GetMaxTaxGrowthPenaltyPercent() != constants.MaxTaxGrowthPenaltyPercent {
		t.Fatalf("population policy = %+v", policy)
	}
	if policy.GetTaxGoldPerPopulation().GetValue() != constants.TaxGoldPerPopPerHour || policy.GetTaxGoldPerPopulation().GetScale() != constants.SecondsPerHour {
		t.Fatalf("tax rate = %+v", policy.GetTaxGoldPerPopulation())
	}
}

func TestGameConfigExposesBarracksTrainingSpeedByLevel(t *testing.T) {
	configs := buildBuildingConfigs()
	for _, config := range configs {
		if config.GetType() != mapping.BuildingTypeToProto(domain.BuildingTypeBarracks) {
			continue
		}
		if len(config.GetLevels()) != constants.MAX_BUILDING_LEVEL {
			t.Fatalf("barracks levels = %d", len(config.GetLevels()))
		}
		for _, level := range config.GetLevels() {
			want := constants.GetBarracksTrainingSpeed(int(level.GetLevel()))
			if level.GetTrainingSpeedMultiplier() != want {
				t.Fatalf("level %d speed = %.2f, want %.2f", level.GetLevel(), level.GetTrainingSpeedMultiplier(), want)
			}
		}
		return
	}
	t.Fatal("barracks config missing")
}
