package database

import (
	"cityio/internal/models"
)

func (c City) ToModel() *models.City {
	return &models.City{
		CityID:        c.CityID,
		Type:          c.Type,
		Owner:         c.Owner,
		Name:          c.Name,
		Population:    c.Population,
		PopulationCap: c.PopulationCap,
		StartX:        c.StartCoords.X,
		StartY:        c.StartCoords.Y,
		Size:          int(c.Size),
	}
}

func (c GetAllCitiesRow) ToModel() *models.City {
	return &models.City{
		CityID:        c.CityID,
		Type:          c.Type,
		Owner:         c.Owner,
		Name:          c.Name,
		Population:    c.Population,
		PopulationCap: c.PopulationCap,
		StartX:        int(c.StartX),
		StartY:        int(c.StartY),
		Size:          int(c.Size),
	}
}

func (u User) ToModel() *models.User {
	return &models.User{
		UserID:    u.UserID,
		Email:     u.Email,
		Username:  u.Username,
		Password:  u.Password,
		Gold:      u.Gold,
		Food:      u.Food,
		CreatedAt: u.CreatedAt.Time,
		UpdatedAt: u.UpdatedAt.Time,
	}
}
