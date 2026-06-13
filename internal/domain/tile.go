package domain

// Tile is a single map cell, optionally occupied by a city and/or building.
type Tile struct {
	X          int     `json:"x"`
	Y          int     `json:"y"`
	CityID     *string `json:"cityId"`
	BuildingID *string `json:"buildingId"`
}
