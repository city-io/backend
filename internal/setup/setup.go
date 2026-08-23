// Package setup initializes the application state by restoring actors from the database.
package setup

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"cityio/internal/constants"
	"cityio/internal/contracts"
	"cityio/internal/database"
	"cityio/internal/domain"
	"cityio/internal/logger"
	"cityio/internal/services"
	"cityio/internal/worldgen"
)

type Deps struct {
	DB      database.Querier
	Cluster contracts.ClusterProvider
	World   *worldgen.World
}

func Run(ctx context.Context, deps *Deps) {
	ctx = logger.With(ctx, "phase", "init")
	db := deps.DB
	cluster := deps.Cluster

	users, err := db.GetAllUsers(ctx)
	if err != nil {
		panic(err)
	}
	cities, err := db.GetAllCities(ctx)
	if err != nil {
		panic(err)
	}
	armies, err := db.GetAllArmies(ctx)
	if err != nil {
		panic(err)
	}
	buildings, err := db.GetAllBuildings(ctx)
	if err != nil {
		panic(err)
	}

	databaseEmpty := len(users) == 0 && len(cities) == 0 && len(armies) == 0 && len(buildings) == 0
	if databaseEmpty {
		if err := seedWorld(ctx, deps); err != nil {
			panic(err)
		}
		cities, err = db.GetAllCities(ctx)
		if err != nil {
			panic(err)
		}
		buildings, err = db.GetAllBuildings(ctx)
		if err != nil {
			panic(err)
		}
	} else {
		slog.InfoContext(ctx, "preserving existing world state",
			"users", len(users),
			"cities", len(cities),
			"buildings", len(buildings),
			"armies", len(armies),
		)
	}
	for _, city := range cities {
		deps.World.RestoreSettlement(*city.ToModel())
	}

	for _, user := range users {
		err := services.RestoreUser(ctx, cluster, user.ToModel())
		if err != nil {
			panic(err)
		}
	}
	slog.InfoContext(ctx, "spawned user actors", "count", len(users))

	for _, city := range cities {
		err := services.RestoreCity(ctx, cluster, city.ToModel())
		if err != nil {
			panic(err)
		}
	}
	slog.InfoContext(ctx, "spawned city actors", "count", len(cities))

	for _, army := range armies {
		err := services.RestoreArmy(ctx, cluster, army.ToModel())
		if err != nil {
			panic(err)
		}
	}
	slog.InfoContext(ctx, "spawned army actors", "count", len(armies))

	for _, building := range buildings {
		err := services.RestoreBuilding(ctx, cluster, building.ToModel())
		if err != nil {
			panic(err)
		}
	}
	slog.InfoContext(ctx, "spawned building actors", "count", len(buildings))

	// Create the test user AFTER the bulk restore only when it does not already
	// exist. Recreating it would violate the unique email constraint now that
	// development restarts preserve the database.
	// TODO: remove test user registration once real registration is the only
	// path.
	testUserExists := false
	for _, user := range users {
		if user.Email == "cityio@example.com" || user.Username == "cityio" {
			testUserExists = true
			break
		}
	}
	if !testUserExists {
		userID, err := services.CreateUser(ctx, cluster, &services.CreateUserRequest{
			Email:    "cityio@example.com",
			Username: "cityio",
			Password: "cityio",
		})
		if err != nil {
			panic(err)
		}
		slog.InfoContext(ctx, "registered test user", "user_id", userID)
	}

	slog.InfoContext(ctx, "initialization complete")
}

func seedWorld(ctx context.Context, deps *Deps) error {
	ctx = logger.With(ctx, "phase", "seed")
	db := deps.DB

	townPlans := deps.World.Towns()
	cities := make([]domain.City, 0, len(townPlans))
	buildings := make([]domain.Building, 0, len(townPlans)*2)
	for _, town := range townPlans {
		cityID := uuid.New().String()
		var populationCap float64
		for _, building := range town.Buildings {
			populationCap += constants.GetBuildingPopulation(building.Type, building.Level)
		}

		city := domain.City{
			CityID:          cityID,
			Type:            domain.CityTypeTown,
			Owner:           nil,
			Name:            town.Name,
			Population:      populationCap,
			PopulationCap:   populationCap,
			GarrisonPercent: constants.NeutralGarrisonPercent,
			TaxRatePercent:  0,
			StartX:          town.X,
			StartY:          town.Y,
			Size:            town.Size,
		}
		city.GarrisonPopulation = constants.GarrisonTarget(city)
		cities = append(cities, city)
		for _, building := range town.Buildings {
			buildings = append(buildings, domain.Building{
				BuildingID:        uuid.New().String(),
				CityID:            cityID,
				Type:              string(building.Type),
				Level:             building.Level,
				TargetLevel:       building.Level,
				X:                 building.X,
				Y:                 building.Y,
				ConstructionStart: domain.NullTime{Time: nil},
				ConstructionEnd:   domain.NullTime{Time: nil},
			})
		}
	}

	cityBatchSize := 5000
	for i := 0; i < len(cities); i += cityBatchSize {
		end := min(i+cityBatchSize, len(cities))
		chunk := cities[i:end]

		params := database.BatchCreateCitiesParams{
			CityIds:             make([]string, 0, len(chunk)),
			Types:               make([]string, 0, len(chunk)),
			Owners:              make([]string, 0, len(chunk)),
			Names:               make([]string, 0, len(chunk)),
			Populations:         make([]float64, 0, len(chunk)),
			PopulationCaps:      make([]float64, 0, len(chunk)),
			GarrisonPopulations: make([]float64, 0, len(chunk)),
			GarrisonPercents:    make([]int32, 0, len(chunk)),
			TaxRatePercents:     make([]int32, 0, len(chunk)),
			StartXs:             make([]int32, 0, len(chunk)),
			StartYs:             make([]int32, 0, len(chunk)),
			Sizes:               make([]int32, 0, len(chunk)),
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
			params.GarrisonPopulations = append(params.GarrisonPopulations, city.GarrisonPopulation)
			params.GarrisonPercents = append(params.GarrisonPercents, int32(city.GarrisonPercent))
			params.TaxRatePercents = append(params.TaxRatePercents, int32(city.TaxRatePercent))
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
			params.Xs = append(params.Xs, int32(b.X))
			params.Ys = append(params.Ys, int32(b.Y))
			params.ConstructionStarts = append(params.ConstructionStarts, database.ToPGTimestamp(b.ConstructionStart.Time))
			params.ConstructionEnds = append(params.ConstructionEnds, database.ToPGTimestamp(b.ConstructionEnd.Time))
		}

		if err := db.BatchCreateBuildings(ctx, params); err != nil {
			slog.ErrorContext(ctx, "error batch creating buildings", "start_idx", i, "end_idx", end, "error", err)
			return err
		}
	}

	slog.DebugContext(ctx, "world seed complete")
	return nil
}
