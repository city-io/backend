// Package models contains data models used across the application.
package models

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type CityType string
type BuildingType string
type NullTime struct {
	*time.Time
}

func (n NullTime) ToPG() pgtype.Timestamp {
	if n.Time == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *n.Time, Valid: true}
}

type Coordinates struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type User struct {
	UserID   string `json:"userId"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	Gold     int64  `json:"gold"`
	Food     int64  `json:"food"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type Tile struct {
	X          int     `json:"x"`
	Y          int     `json:"y"`
	CityID     *string `json:"cityId"`
	BuildingID *string `json:"buildingId"`
}

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

func (b Building) BuildingType() BuildingType {
	return BuildingType(b.Type)
}
