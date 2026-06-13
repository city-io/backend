package domain

import "time"

// BuildingType identifies a kind of building and its behavior.
type BuildingType string

const (
	BuildingTypeCityCenter BuildingType = "city_center"
	BuildingTypeTownCenter BuildingType = "town_center"
	BuildingTypeBarracks   BuildingType = "barracks"
	BuildingTypeHouse      BuildingType = "house"
	BuildingTypeFarm       BuildingType = "farm"
	BuildingTypeMine       BuildingType = "mine"
)

// Building is a structure within a city.
type Building struct {
	BuildingID        string    `json:"building_id"`
	CityID            string    `json:"city_id"`
	Type              string    `json:"type"`
	Level             int       `json:"level"`
	TargetLevel       int       `json:"target_level"`
	X                 int       `json:"x"`
	Y                 int       `json:"y"`
	ConstructionStart NullTime  `json:"construction_start"`
	ConstructionEnd   NullTime  `json:"construction_end"`
	CreatedAt         time.Time `json:"-"`
	UpdatedAt         time.Time `json:"-"`
}

// BuildingType returns the typed building kind.
func (b Building) BuildingType() BuildingType {
	return BuildingType(b.Type)
}
