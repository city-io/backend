// Package setup initializes the application state by restoring actors from the database.
package setup

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"cityio/internal/constants"
	"cityio/internal/database"
	"cityio/internal/logger"
	"cityio/internal/models"
	"cityio/internal/ports"
	"cityio/internal/services"
)

type Deps struct {
	DB      database.Querier
	Cluster ports.ClusterProvider
}

func Run(ctx context.Context, deps *Deps) {
	reset(ctx, deps)
	ctx = logger.With(ctx, "phase", "init")
	db := deps.DB
	cluster := deps.Cluster

	users, err := db.GetAllUsers(ctx)
	if err != nil {
		panic(err)
	}

	for _, user := range users {
		err := services.RestoreUser(ctx, cluster, user.ToModel())
		if err != nil {
			panic(err)
		}
	}
	slog.InfoContext(ctx, "spawned user actors", "count", len(users))

	// TODO: remove test user registration later
	userID, err := services.CreateUser(ctx, cluster, &models.CreateUserRequest{
		Email:    "cityio@example.com",
		Username: "cityio",
		Password: "cityio",
	})
	if err != nil {
		panic(err)
	}
	slog.InfoContext(ctx, "registered test user", "user_id", userID)

	// tiles, err := db.GetAllTiles(ctx)
	// if err != nil {
	// 	panic(err)
	// }

	// for _, tile := range tiles {
	// 	err := ctrls.Tile().Restore(tile.ToModel())
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }
	// log.Info("spawned map tile actors", "count", len(tiles))

	cities, err := db.GetAllCities(ctx)
	if err != nil {
		panic(err)
	}

	for _, city := range cities {
		err := services.RestoreCity(ctx, cluster, city.ToModel())
		if err != nil {
			panic(err)
		}
	}
	slog.InfoContext(ctx, "spawned city actors", "count", len(cities))

	// var armies []models.Army
	// cl.DB().Find(&armies)

	// for _, army := range armies {
	// 	err := services.RestoreArmy(army)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }
	// log.Printf("Spawned actors for %d armies", len(armies))

	buildings, err := db.GetAllBuildings(ctx)
	if err != nil {
		panic(err)
	}

	for _, building := range buildings {
		// city and town center will get restored by city actor
		if building.Type == string(constants.BuildingTypeCityCenter) || building.Type == string(constants.BuildingTypeTownCenter) {
			continue
		}
		err := services.RestoreBuilding(ctx, cluster, building.ToModel())
		if err != nil {
			panic(err)
		}
	}
	slog.InfoContext(ctx, "spawned building actors", "count", len(buildings))

	slog.InfoContext(ctx, "initialization complete")
}

func reset(ctx context.Context, deps *Deps) error {
	ctx = logger.With(ctx, "phase", "reset")
	db := deps.DB

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	occupied := make([][]bool, constants.MapSize)
	for i := range occupied {
		occupied[i] = make([]bool, constants.MapSize)
	}

	users, err := db.GetAllUsers(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "error fetching existing users", "error", err)
	}

	for _, user := range users {
		user.Gold = constants.InitialPlayerGold
		user.Food = constants.InitialPlayerFood
		err := db.UpdateUserStats(ctx, database.UpdateUserStatsParams{
			Gold: user.Gold,
			Food: user.Food,
		})
		if err != nil {
			slog.ErrorContext(ctx, "error resetting user fields", "error", err)
		}

		src := rand.NewSource(time.Now().UnixNano())
		r := rand.New(src)
		startX := r.Intn(constants.MapSize - constants.CitySize)
		startY := r.Intn(constants.MapSize - constants.CitySize)

		cityID := uuid.New().String()
		err = db.CreateCity(ctx, database.CreateCityParams{
			CityID:        cityID,
			Type:          "capital",
			Owner:         &user.UserID,
			Name:          fmt.Sprintf("%s's City", user.Username),
			Population:    constants.InitialPlayerCityPopulation,
			PopulationCap: constants.GetBuildingPopulation(constants.BuildingTypeCityCenter, 1),
			StartX:        int32(startX),
			StartY:        int32(startY),
		})
		if err != nil {
			slog.ErrorContext(ctx, "error creating city in db", "error", err)
			return err
		}
		slog.DebugContext(ctx, "created city in db", "city_id", cityID, "user", user.Username, "x", startX, "y", startY)

		err = db.CreateBuilding(ctx, database.CreateBuildingParams{
			BuildingID:        uuid.New().String(),
			CityID:            cityID,
			Type:              string(constants.BuildingTypeCityCenter),
			Level:             1,
			TargetLevel:       1,
			X:                 int32(startX + constants.CitySize/2),
			Y:                 int32(startY + constants.CitySize/2),
			ConstructionStart: pgtype.Timestamp{Valid: false},
			ConstructionEnd:   pgtype.Timestamp{Valid: false},
		})
		if err != nil {
			slog.ErrorContext(ctx, "error creating building in db", "error", err)
			return err
		}

		for i := range constants.CitySize {
			for j := range constants.CitySize {
				occupied[startX+i][startY+j] = true
			}
		}
	}

	cities := make([]models.City, 0)
	buildings := make([]models.Building, 0)
	for x := range constants.MapSize {
		for y := range constants.MapSize {
			open := true
			// TODO: optimize random city placement
			for i := -1; i < 6; i++ {
				for j := -1; j < 6; j++ {
					if x+i < 0 || y+j < 0 || x+i >= constants.MapSize || y+j >= constants.MapSize || occupied[x+i][y+j] {
						open = false
						break
					}
				}
			}

			if open {
				size := 0
				rng := r.Intn(1000)
				if rng < 3 {
					size = 5
				} else if rng < 10 {
					size = 4
				} else if rng < 50 {
					size = 3
				} else if rng < 100 {
					size = 2
				}
				if size > 0 && x+size < constants.MapSize && y+size < constants.MapSize {
					cityID := uuid.New().String()
					cities = append(cities, models.City{
						CityID:        cityID,
						Type:          "town",
						Owner:         nil,
						Name:          fmt.Sprintf("Town %s", cityID),
						Population:    constants.InitialTownPopulation,
						PopulationCap: constants.GetBuildingPopulation(constants.BuildingTypeTownCenter, 1),
						StartX:        x,
						StartY:        y,
						Size:          size,
					})
					buildings = append(buildings, models.Building{
						BuildingID:        uuid.New().String(),
						CityID:            cityID,
						Type:              string(constants.BuildingTypeTownCenter),
						Level:             1,
						TargetLevel:       1,
						X:                 x + size/2,
						Y:                 y + size/2,
						ConstructionStart: models.NullTime{Time: nil},
						ConstructionEnd:   models.NullTime{Time: nil},
					})
					occupied[x][y] = true
					for i := 0; i < size; i++ {
						for j := 0; j < size; j++ {
							occupied[x+i][y+j] = true
						}
					}
				}
			}
		}
	}

	cityBatchSize := 5000
	for i := 0; i < len(cities); i += cityBatchSize {
		end := min(i+cityBatchSize, len(cities))
		chunk := cities[i:end]

		params := database.BatchCreateCitiesParams{
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

		if err := db.BatchCreateCities(ctx, params); err != nil {
			slog.ErrorContext(ctx, "error batch creating cities", "start_idx", i, "end_idx", end, "error", err)
			return err
		}
	}

	slog.DebugContext(ctx, "created cities", "count", len(cities))

	buildingBatchSize := 5000
	for i := 0; i < len(buildings); i += buildingBatchSize {
		end := min(i+buildingBatchSize, len(buildings))
		chunk := buildings[i:end]

		params := database.BatchCreateBuildingsParams{
			BuildingIds:        make([]string, 0, len(chunk)),
			CityIds:            make([]string, 0, len(chunk)),
			Types:              make([]string, 0, len(chunk)),
			Levels:             make([]int32, 0, len(chunk)),
			TargetLevels:       make([]int32, 0, len(chunk)),
			Xs:                 make([]int32, 0, len(chunk)),
			Ys:                 make([]int32, 0, len(chunk)),
			ConstructionStarts: make([]pgtype.Timestamp, 0, len(chunk)),
			ConstructionEnds:   make([]pgtype.Timestamp, 0, len(chunk)),
		}

		for _, b := range chunk {
			params.BuildingIds = append(params.BuildingIds, b.BuildingID)
			params.CityIds = append(params.CityIds, b.CityID)
			params.Types = append(params.Types, string(b.Type))
			params.Levels = append(params.Levels, int32(b.Level))
			params.TargetLevels = append(params.TargetLevels, int32(b.TargetLevel))
			params.Xs = append(params.Xs, int32(b.X))
			params.Ys = append(params.Ys, int32(b.Y))
			params.ConstructionStarts = append(params.ConstructionStarts, b.ConstructionStart.ToPG())
			params.ConstructionEnds = append(params.ConstructionEnds, b.ConstructionEnd.ToPG())
		}

		if err := db.BatchCreateBuildings(ctx, params); err != nil {
			slog.ErrorContext(ctx, "error batch creating buildings", "start_idx", i, "end_idx", end, "error", err)
			return err
		}
	}

	slog.DebugContext(ctx, "reset complete")
	return nil
}
