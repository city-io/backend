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
	// PopulationBasis is the highest resident population reached. Recruitment
	// and casualties do not lower it, so housing upgrades or repeated training
	// cannot move the protected civilian floor underneath existing residents.
	PopulationBasis float64 `json:"populationBasis"`
	// MilitiaPopulation is the non-mobile defensive reserve currently present
	// in the settlement. MilitiaTarget is the exact desired defender count;
	// losses refill from future population growth rather than appearing for free.
	MilitiaPopulation float64 `json:"militiaPopulation"`
	MilitiaTarget     float64 `json:"militiaTarget"`
	TaxRatePercent    int     `json:"taxRatePercent"`
	StartX            int     `json:"startX"`
	StartY            int     `json:"startY"`
	Size              int     `json:"size"`

	// FoodProductionRate is the food this city's own farms produce per hour.
	// FoodUpkeep is the food this city's population consumes per hour. NetFoodFlow
	// = production - upkeep (positive = surplus exported to the user pool;
	// negative = imported from it). Starving is true when the city's stable
	// local production rate is below its upkeep rate; pool imports do not make a
	// structurally under-producing city grow.
	FoodProductionRate int64 `json:"foodProductionRate"`
	FoodUpkeep         int64 `json:"foodUpkeep"`
	NetFoodFlow        int64 `json:"netFoodFlow"`
	Starving           bool  `json:"starving"`

	// PopulationGrowthBeforeTaxRate is the current per-hour biological growth
	// baseline. PopulationGrowthRate applies the configured tax modifier to it;
	// both are derived from each tick and are not persisted.
	PopulationGrowthBeforeTaxRate int64   `json:"populationGrowthBeforeTaxRate"`
	PopulationGrowthRate          int64   `json:"populationGrowthRate"`
	TaxIncomeRate                 int64   `json:"taxIncomeRate"`
	MilitiaBattleID               *string `json:"-"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
