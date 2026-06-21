// Package persistence is the write-behind backing store for the actor system.
// Reads, creates and deletes go straight to the database; updates are coalesced
// per entity and flushed in batches by a background goroutine, so the hot
// in-memory actor state is periodically backed up rather than written on every
// tick. It replaces the former single database actor, freeing reads and writes
// to use the connection pool concurrently.
package persistence

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"cityio/internal/constants"
	"cityio/internal/database"
	"cityio/internal/domain"
)

const batchSize = 5000

// ErrNotFound is returned by lookups when no matching row exists.
var ErrNotFound = errors.New("not found")

// Store implements ports.Store over a sqlc Querier backed by a pgx pool.
type Store struct {
	db database.Querier

	mu             sync.Mutex
	userBuffer     map[string]domain.User
	cityBuffer     map[string]domain.City
	buildingBuffer map[string]domain.Building

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

// New constructs a Store. Call Start to begin periodic flushing.
func New(db database.Querier) *Store {
	return &Store{
		db:             db,
		userBuffer:     make(map[string]domain.User),
		cityBuffer:     make(map[string]domain.City),
		buildingBuffer: make(map[string]domain.Building),
		stopTickerCh:   make(chan struct{}),
	}
}

// Start launches the background flush loop. ctx is used for logging context on
// the flush writes.
func (s *Store) Start(ctx context.Context) {
	s.ticker = time.NewTicker(constants.DBBackupFrequency * time.Second)
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.flush(ctx)
			case <-s.stopTickerCh:
				s.ticker.Stop()
				return
			}
		}
	}()
}

// Stop halts the flush loop and performs a final flush so buffered updates are
// not lost on a graceful shutdown.
func (s *Store) Stop(ctx context.Context) {
	select {
	case <-s.stopTickerCh:
	default:
		close(s.stopTickerCh)
	}
	s.flush(ctx)
}

func (s *Store) FindEmptyCityBlock(ctx context.Context, size int) (domain.Coordinates, error) {
	row, err := s.db.FindEmptyCityBlock(ctx, database.FindEmptyCityBlockParams{
		MapWidth:  constants.MapSize,
		MapHeight: constants.MapSize,
		Size:      int32(size),
	})
	if err != nil {
		return domain.Coordinates{}, err
	}
	return domain.Coordinates{X: int(row.X), Y: int(row.Y)}, nil
}

func (s *Store) GetUserByIdentifier(ctx context.Context, identifier string) (*domain.User, error) {
	row, err := s.db.GetUserByIdentifier(ctx, identifier)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.ToModel(), nil
}

func (s *Store) GetAllCities(ctx context.Context) ([]domain.City, error) {
	rows, err := s.db.GetAllCities(ctx)
	if err != nil {
		return nil, err
	}
	cities := make([]domain.City, 0, len(rows))
	for _, c := range rows {
		cities = append(cities, *c.ToModel())
	}
	return cities, nil
}

func (s *Store) GetAllBuildings(ctx context.Context) ([]domain.Building, error) {
	rows, err := s.db.GetAllBuildings(ctx)
	if err != nil {
		return nil, err
	}
	buildings := make([]domain.Building, 0, len(rows))
	for _, b := range rows {
		buildings = append(buildings, *b.ToModel())
	}
	return buildings, nil
}

func (s *Store) GetCitiesByOwner(ctx context.Context, owner string) ([]domain.City, error) {
	rows, err := s.db.GetCitiesByOwner(ctx, &owner)
	if err != nil {
		return nil, err
	}
	cities := make([]domain.City, 0, len(rows))
	for _, c := range rows {
		cities = append(cities, *c.ToModel())
	}
	return cities, nil
}

func (s *Store) GetBuildingsByCity(ctx context.Context, cityID string) ([]domain.Building, error) {
	rows, err := s.db.GetBuildingsByCity(ctx, cityID)
	if err != nil {
		return nil, err
	}
	buildings := make([]domain.Building, 0, len(rows))
	for _, b := range rows {
		buildings = append(buildings, *b.ToModel())
	}
	return buildings, nil
}

func (s *Store) CreateUser(ctx context.Context, user domain.User) error {
	return s.db.CreateUser(ctx, database.CreateUserParams{
		UserID:   user.UserID,
		Email:    user.Email,
		Username: user.Username,
		Password: user.Password,
	})
}

func (s *Store) CreateCity(ctx context.Context, city domain.City) error {
	return s.db.CreateCity(ctx, database.CreateCityParams{
		CityID:        city.CityID,
		Type:          string(city.Type),
		Owner:         city.Owner,
		Name:          city.Name,
		Population:    city.Population,
		PopulationCap: city.PopulationCap,
		StartX:        int32(city.StartX),
		StartY:        int32(city.StartY),
		Size:          int32(city.Size),
	})
}

func (s *Store) CreateBuilding(ctx context.Context, building domain.Building) error {
	return s.db.CreateBuilding(ctx, database.CreateBuildingParams{
		BuildingID:        building.BuildingID,
		CityID:            building.CityID,
		Type:              building.Type,
		Level:             int32(building.Level),
		X:                 int32(building.X),
		Y:                 int32(building.Y),
		ConstructionStart: database.ToPGTimestamp(building.ConstructionStart.Time),
		ConstructionEnd:   database.ToPGTimestamp(building.ConstructionEnd.Time),
	})
}

func (s *Store) DeleteUser(ctx context.Context, userID string) error {
	s.mu.Lock()
	delete(s.userBuffer, userID)
	s.mu.Unlock()
	return s.db.DeleteUser(ctx, userID)
}

func (s *Store) DeleteCity(ctx context.Context, cityID string) error {
	s.mu.Lock()
	delete(s.cityBuffer, cityID)
	s.mu.Unlock()
	return s.db.DeleteCity(ctx, cityID)
}

func (s *Store) DeleteBuilding(ctx context.Context, buildingID string) error {
	s.mu.Lock()
	delete(s.buildingBuffer, buildingID)
	s.mu.Unlock()
	return s.db.DeleteBuilding(ctx, buildingID)
}

func (s *Store) EnqueueUser(user domain.User) {
	s.mu.Lock()
	s.userBuffer[user.UserID] = user
	s.mu.Unlock()
}

func (s *Store) EnqueueCity(city domain.City) {
	s.mu.Lock()
	s.cityBuffer[city.CityID] = city
	s.mu.Unlock()
}

func (s *Store) EnqueueBuilding(building domain.Building) {
	s.mu.Lock()
	s.buildingBuffer[building.BuildingID] = building
	s.mu.Unlock()
}

// flush swaps out the pending buffers under the lock, then writes the snapshots
// without holding it so enqueues continue while a flush is in flight.
func (s *Store) flush(ctx context.Context) {
	s.mu.Lock()
	users := s.userBuffer
	cities := s.cityBuffer
	buildings := s.buildingBuffer
	s.userBuffer = make(map[string]domain.User)
	s.cityBuffer = make(map[string]domain.City)
	s.buildingBuffer = make(map[string]domain.Building)
	s.mu.Unlock()

	s.flushCities(ctx, cities)
	s.flushUsers(ctx, users)
	s.flushBuildings(ctx, buildings)
}

func (s *Store) flushCities(ctx context.Context, buffer map[string]domain.City) {
	cities := make([]domain.City, 0, len(buffer))
	for _, c := range buffer {
		cities = append(cities, c)
	}
	for i := 0; i < len(cities); i += batchSize {
		end := min(i+batchSize, len(cities))
		chunk := cities[i:end]

		params := database.BatchUpdateCitiesParams{
			CityIds:        make([]string, 0, len(chunk)),
			Types:          make([]string, 0, len(chunk)),
			Owners:         make([]string, 0, len(chunk)),
			Names:          make([]string, 0, len(chunk)),
			Populations:    make([]float64, 0, len(chunk)),
			PopulationCaps: make([]float64, 0, len(chunk)),
			StartXs:        make([]int32, 0, len(chunk)),
			StartYs:        make([]int32, 0, len(chunk)),
			Sizes:          make([]int32, 0, len(chunk)),
		}

		for _, city := range chunk {
			params.CityIds = append(params.CityIds, city.CityID)
			params.Types = append(params.Types, string(city.Type))

			// sqlc will parse "" into NULL
			if city.Owner == nil {
				params.Owners = append(params.Owners, "")
			} else {
				params.Owners = append(params.Owners, *city.Owner)
			}

			params.Names = append(params.Names, city.Name)
			params.Populations = append(params.Populations, city.Population)
			params.PopulationCaps = append(params.PopulationCaps, city.PopulationCap)
			params.StartXs = append(params.StartXs, int32(city.StartX))
			params.StartYs = append(params.StartYs, int32(city.StartY))
			params.Sizes = append(params.Sizes, int32(city.Size))
		}

		if err := s.db.BatchUpdateCities(ctx, params); err != nil {
			slog.ErrorContext(ctx, "error batch updating cities", "idx", i, "error", err)
		}
	}
}

func (s *Store) flushUsers(ctx context.Context, buffer map[string]domain.User) {
	users := make([]domain.User, 0, len(buffer))
	for _, u := range buffer {
		users = append(users, u)
	}
	for i := 0; i < len(users); i += batchSize {
		end := min(i+batchSize, len(users))
		chunk := users[i:end]

		params := database.BatchUpdateUsersParams{
			UserIds: make([]string, 0, len(chunk)),
			Foods:   make([]int64, 0, len(chunk)),
			Golds:   make([]int64, 0, len(chunk)),
		}

		for _, user := range chunk {
			params.UserIds = append(params.UserIds, user.UserID)
			params.Foods = append(params.Foods, user.Food)
			params.Golds = append(params.Golds, user.Gold)
		}

		if err := s.db.BatchUpdateUsers(ctx, params); err != nil {
			slog.ErrorContext(ctx, "error batch updating users", "idx", i, "error", err)
		}
	}
}

func (s *Store) flushBuildings(ctx context.Context, buffer map[string]domain.Building) {
	buildings := make([]domain.Building, 0, len(buffer))
	for _, b := range buffer {
		buildings = append(buildings, b)
	}
	for i := 0; i < len(buildings); i += batchSize {
		end := min(i+batchSize, len(buildings))
		chunk := buildings[i:end]

		params := database.BatchUpdateBuildingsParams{
			BuildingIds:        make([]string, 0, len(chunk)),
			CityIds:            make([]string, 0, len(chunk)),
			Types:              make([]string, 0, len(chunk)),
			Levels:             make([]int32, 0, len(chunk)),
			Xs:                 make([]int32, 0, len(chunk)),
			Ys:                 make([]int32, 0, len(chunk)),
			ConstructionStarts: make([]pgtype.Timestamp, 0, len(chunk)),
			ConstructionEnds:   make([]pgtype.Timestamp, 0, len(chunk)),
		}

		for _, b := range chunk {
			params.BuildingIds = append(params.BuildingIds, b.BuildingID)
			params.CityIds = append(params.CityIds, b.CityID)
			params.Types = append(params.Types, b.Type)
			params.Levels = append(params.Levels, int32(b.Level))
			params.Xs = append(params.Xs, int32(b.X))
			params.Ys = append(params.Ys, int32(b.Y))
			params.ConstructionStarts = append(params.ConstructionStarts, database.ToPGTimestamp(b.ConstructionStart.Time))
			params.ConstructionEnds = append(params.ConstructionEnds, database.ToPGTimestamp(b.ConstructionEnd.Time))
		}

		if err := s.db.BatchUpdateBuildings(ctx, params); err != nil {
			slog.ErrorContext(ctx, "error batch updating buildings", "idx", i, "error", err)
		}
	}
}
