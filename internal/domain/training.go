package domain

import "time"

// TrainingOrder is a paid troop batch queued at a barracks.
type TrainingOrder struct {
	TrainingOrderID string
	ArmyID          string
	BarracksID      string
	TroopType       TroopType
	Count           int64
	PopulationCost  int64
	GoldCost        int64
	StartedAt       NullTime
	CompletesAt     NullTime
	CreatedAt       time.Time
}
