package domain

import "time"

// TrainingOrder is a paid troop batch in a city's shared training pipeline.
// BarracksID and the timing fields are empty until a barracks claims it.
type TrainingOrder struct {
	TrainingOrderID string
	ArmyID          string
	CityID          string
	BarracksID      *string
	TroopType       TroopType
	Count           int64
	PopulationCost  int64
	GoldCost        int64
	StartedAt       NullTime
	CompletesAt     NullTime
	CreatedAt       time.Time
}
