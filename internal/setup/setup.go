// Package setup initializes the application state by restoring actors from the database.
package setup

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"cityio/internal/constants"
	"cityio/internal/database"
	"cityio/internal/models"
	"cityio/internal/ports"
)

type Deps struct {
	Log         ports.Logger
	DB          database.Querier
	Controllers ports.Controllers
}

func Run(deps *Deps) {
	reset(deps)
	log := deps.Log.With("phase", "init")
	db := deps.DB
	ctrls := deps.Controllers
	ctx := context.Background()

	users, err := db.GetAllUsers(ctx)
	if err != nil {
		panic(err)
	}

	for _, user := range users {
		err := ctrls.User().Restore(user.ToModel())
		if err != nil {
			panic(err)
		}
	}
	log.Info("spawned user actors", "count", len(users))

	// TODO: remove test user registration later
	userID, err := ctrls.User().Create(&models.CreateUserRequest{
		Email:    "cityio@example.com",
		Username: "cityio",
		Password: "cityio",
	})
	if err != nil {
		panic(err)
	}
	log.Info("registered test user", "user_id", userID)

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
		err := ctrls.City().Restore(city.ToModel())
		if err != nil {
			panic(err)
		}
	}
	log.Info("spawned city actors", "count", len(cities))

	// var armies []models.Army
	// cl.DB().Find(&armies)

	// for _, army := range armies {
	// 	err := services.RestoreArmy(army)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }
	// log.Printf("Spawned actors for %d armies", len(armies))

	// var buildings []models.Building
	// cl.DB().Find(&buildings)

	// for _, building := range buildings {
	// 	err := services.RestoreBuilding(building)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }
	// log.Printf("Spawned actors for %d buildings", len(buildings))
	log.Info("initialization complete")
}

func reset(deps *Deps) error {
	log := deps.Log.With("phase", "reset")
	db := deps.DB

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	occupied := make([][]bool, constants.MapSize)
	for i := range occupied {
		occupied[i] = make([]bool, constants.MapSize)
	}

	users, err := db.GetAllUsers(context.Background())
	if err != nil {
		log.Error("error fetching existing users", "error", err)
	}

	for _, user := range users {
		user.Gold = constants.InitialPlayerGold
		user.Food = constants.InitialPlayerFood
		err := db.UpdateUserStats(context.Background(), database.UpdateUserStatsParams{
			Gold: user.Gold,
			Food: user.Food,
		})
		if err != nil {
			log.Error("error resetting user fields", "error", err)
		}

		src := rand.NewSource(time.Now().UnixNano())
		r := rand.New(src)
		startX := r.Intn(constants.MapSize - constants.CitySize)
		startY := r.Intn(constants.MapSize - constants.CitySize)

		cityID := uuid.New().String()
		err = db.CreateCity(context.Background(), database.CreateCityParams{
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
			log.Error("error creating city in db", "error", err)
			return err
		}
		log.Debug("created city in db", "city_id", cityID, "user", user.Username, "x", startX, "y", startY)

		// result = db.Create(&models.Building{
		// 	BuildingID: uuid.New().String(),
		// 	CityID:     cityID,
		// 	Type:       "city_center",
		// 	Level:      1,
		// 	X:          startX + int(math.Floor(float64(constants.CitySize)/2)),
		// 	Y:          startY + int(math.Floor(float64(constants.CitySize)/2)),
		// })
		// if result.Error != nil {
		// 	log.Error("Error creating building in db", "error", result.Error)
		// 	return result.Error
		// }

		for i := range constants.CitySize {
			for j := range constants.CitySize {
				occupied[startX+i][startY+j] = true
			}
		}
	}

	cities := make([]models.City, 0)
	// buildings := make([]models.Building, 0)
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
					// buildings = append(buildings, models.Building{
					// 	BuildingID: uuid.New().String(),
					// 	CityID:     cityID,
					// 	Type:       "town_center",
					// 	Level:      1,
					// 	X:          x + int(math.Floor(float64(size)/2)),
					// 	Y:          y + int(math.Floor(float64(size)/2)),
					// })
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

		if err := db.BatchCreateCities(context.Background(), params); err != nil {
			log.Error("error batch creating cities", "start_idx", i, "end_idx", end, "error", err)
			return err
		}
	}

	log.Debug("created cities", "count", len(cities))

	// buildingBatchSize := 5000
	// for i := 0; i < len(buildings); i += buildingBatchSize {
	// 	end := min(i+buildingBatchSize, len(buildings))
	// 	if result := db.Create(buildings[i:end]); result.Error != nil {
	// 		log.Error("Error creating buildings", "error", result.Error)
	// 		return result.Error
	// 	}
	// }
	// log.Debug("Created buildings", "count", len(buildings))
	log.Debug("reset complete")
	return nil
}
