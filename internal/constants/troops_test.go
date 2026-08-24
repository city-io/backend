package constants

import (
	"testing"
	"time"

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

func TestBarracksLevelsIncreaseTrainingThroughput(t *testing.T) {
	if got := GetBarracksTrainingSpeed(1); got != 1 {
		t.Fatalf("level 1 speed = %.2f, want 1.00", got)
	}
	if got := GetBarracksTrainingSpeed(5); got != 1.8 {
		t.Fatalf("level 5 speed = %.2f, want 1.80", got)
	}
	base := 50 * time.Second
	if got := GetBarracksTrainingDuration(domain.TroopTypeSoldier, 10, 1); got != base {
		t.Fatalf("level 1 duration = %s, want %s", got, base)
	}
	if got := GetBarracksTrainingDuration(domain.TroopTypeSoldier, 10, 5); got >= base {
		t.Fatalf("level 5 duration = %s, want less than %s", got, base)
	}
}
