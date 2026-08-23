package actors

import (
	"testing"

	"cityio/internal/domain"
)

func TestBuildingProductionRateDuringConstructionAndUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		building domain.Building
		want     int64
	}{
		{
			name:     "new construction is offline",
			building: domain.Building{Type: string(domain.BuildingTypeFarm), Level: 0, TargetLevel: 1},
			want:     0,
		},
		{
			name:     "completed building is at full capacity",
			building: domain.Building{Type: string(domain.BuildingTypeFarm), Level: 1, TargetLevel: 1},
			want:     12000,
		},
		{
			name:     "upgrade keeps current level at seventy five percent",
			building: domain.Building{Type: string(domain.BuildingTypeFarm), Level: 1, TargetLevel: 2},
			want:     9000,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := buildingActor{Building: test.building}
			if got := state.currentProductionRate("food"); got != test.want {
				t.Fatalf("production rate = %d, want %d", got, test.want)
			}
		})
	}
}

func TestUpgradeProductionCarriesFractionalTickOutput(t *testing.T) {
	state := buildingActor{Building: domain.Building{
		Type: string(domain.BuildingTypeCityCenter), Level: 1, TargetLevel: 2,
	}}
	var produced int64
	for range 20 {
		produced += state.productionForTick("gold")
	}
	// 75% of 3,600/hr is 2,700/hr, which produces exactly 45 over 60s.
	if produced != 45 {
		t.Fatalf("production over 60s = %d, want 45", produced)
	}
}
