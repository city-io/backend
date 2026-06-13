package database

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"cityio/internal/domain"
)

// ToPGTimestamp converts an optional time into a pgx timestamp, keeping
// pgx-specific concerns out of the domain layer.
func ToPGTimestamp(t *time.Time) pgtype.Timestamp {
	if t == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *t, Valid: true}
}

func toNullTime(ts pgtype.Timestamp) domain.NullTime {
	if !ts.Valid {
		return domain.NullTime{}
	}
	return domain.NullTime{Time: &ts.Time}
}

func (c City) ToModel() *domain.City {
	return &domain.City{
		CityID:        c.CityID,
		Type:          domain.CityType(c.Type),
		Owner:         c.Owner,
		Name:          c.Name,
		Population:    c.Population,
		PopulationCap: c.PopulationCap,
		StartX:        c.StartCoords.X,
		StartY:        c.StartCoords.Y,
		Size:          int(c.Size),
	}
}

func (c GetAllCitiesRow) ToModel() *domain.City {
	return &domain.City{
		CityID:        c.CityID,
		Type:          domain.CityType(c.Type),
		Owner:         c.Owner,
		Name:          c.Name,
		Population:    c.Population,
		PopulationCap: c.PopulationCap,
		StartX:        int(c.StartX),
		StartY:        int(c.StartY),
		Size:          int(c.Size),
	}
}

func (u User) ToModel() *domain.User {
	return &domain.User{
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

func (b Building) ToModel() *domain.Building {
	return &domain.Building{
		BuildingID:        b.BuildingID,
		CityID:            b.CityID,
		Type:              b.Type,
		Level:             int(b.Level),
		TargetLevel:       int(b.TargetLevel),
		X:                 b.Coords.X,
		Y:                 b.Coords.Y,
		ConstructionStart: toNullTime(b.ConstructionStart),
		ConstructionEnd:   toNullTime(b.ConstructionEnd),
	}
}

func (c GetCitiesByOwnerRow) ToModel() *domain.City {
	return &domain.City{
		CityID:        c.CityID,
		Type:          domain.CityType(c.Type),
		Owner:         c.Owner,
		Name:          c.Name,
		Population:    c.Population,
		PopulationCap: c.PopulationCap,
		StartX:        int(c.StartX),
		StartY:        int(c.StartY),
		Size:          int(c.Size),
	}
}

func (b GetBuildingsByCityRow) ToModel() *domain.Building {
	return &domain.Building{
		BuildingID:        b.BuildingID,
		CityID:            b.CityID,
		Type:              b.Type,
		Level:             int(b.Level),
		TargetLevel:       int(b.TargetLevel),
		X:                 int(b.X),
		Y:                 int(b.Y),
		ConstructionStart: toNullTime(b.ConstructionStart),
		ConstructionEnd:   toNullTime(b.ConstructionEnd),
	}
}

func (b GetAllBuildingsRow) ToModel() *domain.Building {
	return &domain.Building{
		BuildingID:        b.BuildingID,
		CityID:            b.CityID,
		Type:              b.Type,
		Level:             int(b.Level),
		TargetLevel:       int(b.TargetLevel),
		X:                 int(b.X),
		Y:                 int(b.Y),
		ConstructionStart: toNullTime(b.ConstructionStart),
		ConstructionEnd:   toNullTime(b.ConstructionEnd),
	}
}
