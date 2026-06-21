package domain

import "time"

// CityType distinguishes player capitals from neutral towns.
type CityType string

const (
	CityTypeCity CityType = "city"
	CityTypeTown CityType = "town"
)

// City is a settlement on the map, owned by a player or neutral.
type City struct {
	CityID        string   `json:"cityId"`
	Type          CityType `json:"type"`
	Owner         *string  `json:"owner"`
	Name          string   `json:"name"`
	Population    float64  `json:"population"`
	PopulationCap float64  `json:"populationCap"`
	StartX        int      `json:"startX"`
	StartY        int      `json:"startY"`
	Size          int      `json:"size"`

	// FoodProductionRate is the food this city's own farms produce per hour.
	// FoodUpkeep is the food this city's population consumes per hour. NetFoodFlow
	// = production - upkeep (positive = surplus exported to the user pool;
	// negative = imported from it). Starving is true when the city could not
	// cover its demand even after drawing from the pool.
	FoodProductionRate int64 `json:"foodProductionRate"`
	FoodUpkeep         int64 `json:"foodUpkeep"`
	NetFoodFlow        int64 `json:"netFoodFlow"`
	Starving           bool  `json:"starving"`

	// PopulationGrowthRate is the current per-hour population change. Positive
	// when the city is growing, negative when starving and population is
	// declining. Computed from the per-tick delta applied in growPopulation.
	PopulationGrowthRate int64 `json:"populationGrowthRate"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
