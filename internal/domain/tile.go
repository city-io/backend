package domain

// Tile is a single map cell with immutable terrain and mutable occupancy.
type Tile struct {
	X          int         `json:"x"`
	Y          int         `json:"y"`
	Terrain    TerrainType `json:"terrain"`
	CityID     *string     `json:"cityId"`
	BuildingID *string     `json:"buildingId"`
	ArmyIDs    []string    `json:"armyIds"`
}
