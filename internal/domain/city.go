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

	// FoodProductionRate is food per second this city's own farms produced in
	// the last tick. FoodUpkeep is food per second this city's population
	// consumed. NetFoodFlow = production - upkeep (positive = surplus exported
	// to the user pool; negative = imported from it). Starving is true when
	// the city could not cover its demand even after drawing from the pool.
	FoodProductionRate float64 `json:"foodProductionRate"`
	FoodUpkeep         float64 `json:"foodUpkeep"`
	NetFoodFlow        float64 `json:"netFoodFlow"`
	Starving           bool    `json:"starving"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
