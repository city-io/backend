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
	CityID        string    `json:"cityId"`
	Type          CityType  `json:"type"`
	Owner         *string   `json:"owner"`
	Name          string    `json:"name"`
	Population    float64   `json:"population"`
	PopulationCap float64   `json:"populationCap"`
	StartX        int       `json:"startX"`
	StartY        int       `json:"startY"`
	Size          int       `json:"size"`
	CreatedAt     time.Time `json:"-"`
	UpdatedAt     time.Time `json:"-"`
}
