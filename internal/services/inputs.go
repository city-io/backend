package services

import "cityio/internal/domain"

// CreateUserRequest is the command to register a new user.
type CreateUserRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CityInput is the command to create a new city.
type CityInput struct {
	Type  domain.CityType `json:"type"`
	Owner *string         `json:"owner"`
	Name  string          `json:"name"`
	Size  int             `json:"size"`
}

// BuildingInput is the command to construct a new building.
type BuildingInput struct {
	CityID string              `json:"city_id"`
	Type   domain.BuildingType `json:"type"`
	X      int                 `json:"x"`
	Y      int                 `json:"y"`
}
