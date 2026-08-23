package constants

import (
	"testing"

	"cityio/internal/domain"
)

func TestTroopTrainingDurationScalesWithBatchSize(t *testing.T) {
	tests := []struct {
		troopType domain.TroopType
		perTroop  int64
	}{
		{domain.TroopTypeSoldier, 5},
		{domain.TroopTypeArcher, 7},
		{domain.TroopTypeCavalry, 10},
		{domain.TroopTypeArtillery, 15},
	}

	for _, test := range tests {
		if got := GetTroopTrainTime(test.troopType); got != test.perTroop {
			t.Errorf("%s train time = %d, want %d", test.troopType, got, test.perTroop)
		}
		if got := GetTroopTrainingDuration(test.troopType, 5); got != test.perTroop*5 {
			t.Errorf("%s batch duration = %d, want %d", test.troopType, got, test.perTroop*5)
		}
	}
}
