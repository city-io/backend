package ports

import (
	"context"

	"cityio/internal/domain"
)

// Store is the persistence port. Reads, creates and deletes hit the database
// immediately; updates are coalesced per entity (latest-write-wins) and flushed
// in batches by a background writer, so the hot in-memory state is backed up
// without a write per tick.
type Store interface {
	FindEmptyCityBlock(ctx context.Context, size int) (domain.Coordinates, error)
	GetUserByIdentifier(ctx context.Context, identifier string) (*domain.User, error)
	GetAllCities(ctx context.Context) ([]domain.City, error)
	GetAllBuildings(ctx context.Context) ([]domain.Building, error)
	GetCitiesByOwner(ctx context.Context, owner string) ([]domain.City, error)
	GetBuildingsByCity(ctx context.Context, cityID string) ([]domain.Building, error)

	CreateUser(ctx context.Context, user domain.User) error
	CreateCity(ctx context.Context, city domain.City) error
	CreateBuilding(ctx context.Context, building domain.Building) error

	DeleteUser(ctx context.Context, userID string) error
	DeleteCity(ctx context.Context, cityID string) error
	DeleteBuilding(ctx context.Context, buildingID string) error

	EnqueueUser(user domain.User)
	EnqueueCity(city domain.City)
	EnqueueBuilding(building domain.Building)
}
