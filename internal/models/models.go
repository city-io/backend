// Package models contains data models used across the application.
package models

import (
	"time"
)

type BuildingType string

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

type MapTile struct {
	X          int    `json:"x"`
	Y          int    `json:"y"`
	CityID     string `json:"cityId"`
	BuildingID string `json:"buildingId"`
}

type City struct {
	CityID        string    `json:"cityId"`
	Type          string    `json:"type"`
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
