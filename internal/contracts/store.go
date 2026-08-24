package contracts

import (
	"context"
	"errors"
	"time"

	"cityio/internal/domain"
)

var ErrArmyNameTaken = errors.New("army name already exists")

// Store is the persistence contract. Reads, creates and deletes hit the database
// immediately; updates are coalesced per entity (latest-write-wins) and flushed
// in batches by a background writer, so the hot in-memory state is backed up
// without a write per tick.
type Store interface {
	GetUserByIdentifier(ctx context.Context, identifier string) (*domain.User, error)
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	GetAllUsers(ctx context.Context) ([]domain.User, error)
	GetAllCities(ctx context.Context) ([]domain.City, error)
	GetAllBuildings(ctx context.Context) ([]domain.Building, error)
	GetAllArmies(ctx context.Context) ([]domain.Army, error)
	GetTrainingOrdersByCity(ctx context.Context, cityID string) ([]domain.TrainingOrder, error)
	GetMailboxMessagesByRecipient(ctx context.Context, userID string) ([]domain.MailboxMessage, error)
	GetCitiesByOwner(ctx context.Context, owner string) ([]domain.City, error)
	GetBuildingsByCity(ctx context.Context, cityID string) ([]domain.Building, error)
	GetExploredTiles(ctx context.Context, userID string) ([]domain.Coordinates, error)
	AddExploredTiles(ctx context.Context, userID string, tiles []domain.Coordinates) error

	CreateUser(ctx context.Context, user domain.User) error
	CreateCity(ctx context.Context, city domain.City) error
	CreateBuilding(ctx context.Context, building domain.Building) error
	CreateArmy(ctx context.Context, army domain.Army) error
	CreateTrainingOrder(ctx context.Context, order domain.TrainingOrder) error
	CreateMailboxMessage(ctx context.Context, message domain.MailboxMessage) error
	AssignTrainingOrder(ctx context.Context, orderID, barracksID string, startedAt, completesAt time.Time) error
	MarkMailboxMessageRead(ctx context.Context, messageID, userID string) (*domain.MailboxMessage, error)
	RenameArmy(ctx context.Context, armyID, owner, name string) error

	DeleteUser(ctx context.Context, userID string) error
	DeleteCity(ctx context.Context, cityID string) error
	DeleteBuilding(ctx context.Context, buildingID string) error
	DeleteArmy(ctx context.Context, armyID string) error
	DeleteTrainingOrder(ctx context.Context, orderID string) error

	EnqueueUser(user domain.User)
	EnqueueCity(city domain.City)
	EnqueueBuilding(building domain.Building)
	EnqueueArmy(army domain.Army)
}
