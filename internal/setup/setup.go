// Package setup initializes the application state by restoring actors from the database.
package setup

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"

	// "github.com/pressly/goose/v3"

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

	users, err := db.GetAllUsers(context.Background())
	if err != nil {
		panic(err)
	}

	for _, user := range users {
		err := ctrls.User().RestoreUser(user.ToModel())
		if err != nil {
			panic(err)
		}
	}
	log.Info("Spawned user actors", "count", len(users))

	// TODO: Remove test user registration later
	userID, err := ctrls.User().RegisterUser(&models.RegisterUserRequest{
		Email:    "test@email.com",
		Username: "prayujt",
		Password: "test",
	})
	if err != nil {
		panic(err)
	}
	log.Info("Registered test user", "user_id", userID)

	// var mapTiles []models.MapTile
	// cl.DB().Find(&mapTiles)

	// for _, mapTile := range mapTiles {
	// 	err := services.RestoreMapTile(mapTile)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }
	// log.Printf("Spawned actors for %d map tiles", len(mapTiles))

	cities, err := db.GetAllCities(context.Background())
	if err != nil {
		panic(err)
	}

	for _, city := range cities {
		err := ctrls.City().RestoreCity(city.ToModel())
		if err != nil {
			panic(err)
		}
	}
	log.Info("Spawned city actors", "count", len(cities))

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
	log.Info("Initialization complete!")
}

func reset(deps *Deps) error {
	log := deps.Log.With("phase", "reset")
	db := deps.DB

	// resetTable(db, &models.User{})

	// err := resetTable(db, &models.Army{})
	// if err != nil {
	// 	log.Error("Error resetting Army table", "error", err)
	// 	return err
	// }

	// err = resetTable(db, &models.MapTile{})
	// if err != nil {
	// 	log.Error("Error resetting MapTile table", "error", err)
	// 	return err
	// }

	// err = resetTable(db, &models.Building{})
	// if err != nil {
	// 	log.Error("Error resetting Building table", "error", err)
	// 	return err
	// }

	// err = resetTable(db, &models.City{})
	// if err != nil {
	// 	log.Error("Error resetting City table", "error", err)
	// 	return err
	// }

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	occupied := make([][]bool, constants.MapSize)
	for i := range occupied {
		occupied[i] = make([]bool, constants.MapSize)
	}

	users, err := db.GetAllUsers(context.Background())
	if err != nil {
		log.Error("Error fetching existing users", "error", err)
	}

	for _, user := range users {
		user.Gold = constants.InitialPlayerGold
		user.Food = constants.InitialPlayerFood
		err := db.UpdateUserStats(context.Background(), database.UpdateUserStatsParams{
			Gold: user.Gold,
			Food: user.Food,
		})
		if err != nil {
			log.Error("Error resetting user fields", "error", err)
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
			Coordinates:   startX,
			Coordinates_2: startY,
		})
		if err != nil {
			log.Error("Error creating city in db", "error", err)
			return err
		} else {
			log.Debug("Created city in db", "cityId", cityID, "user", user.Username, "x", startX, "y", startY)
		}

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

		for i := 0; i < constants.CitySize; i++ {
			for j := 0; j < constants.CitySize; j++ {
				occupied[startX+i][startY+j] = true
			}
		}
	}

	cities := make([]models.City, 0)
	// buildings := make([]models.Building, 0)
	mapTiles := make([]models.MapTile, 0)
	for x := 0; x < constants.MapSize; x++ {
		for y := 0; y < constants.MapSize; y++ {
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

			mapTiles = append(mapTiles, models.MapTile{
				X: x,
				Y: y,
			})
		}
	}

	// tileBatchSize := 15000
	// for i := 0; i < len(mapTiles); i += tileBatchSize {
	// 	end := min(i+tileBatchSize, len(mapTiles))
	// 	if result := db.Create(mapTiles[i:end]); result.Error != nil {
	// 		log.Error("Error creating map tiles", "error", result.Error)
	// 		return result.Error
	// 	}
	// }
	// log.Debug("Created map tiles", "count", len(mapTiles))

	// cityBatchSize := 5000
	// for i := 0; i < len(cities); i += cityBatchSize {
	// 	end := min(i+cityBatchSize, len(cities))
	// 	if result := db.Create(cities[i:end]); result.Error != nil {
	// 		log.Error("Error creating cities", "error", result.Error)
	// 		return result.Error
	// 	}
	// }
	// log.Debug("Created cities", "count", len(cities))

	// buildingBatchSize := 5000
	// for i := 0; i < len(buildings); i += buildingBatchSize {
	// 	end := min(i+buildingBatchSize, len(buildings))
	// 	if result := db.Create(buildings[i:end]); result.Error != nil {
	// 		log.Error("Error creating buildings", "error", result.Error)
	// 		return result.Error
	// 	}
	// }
	// log.Debug("Created buildings", "count", len(buildings))
	log.Debug("Reset complete!")
	return nil
}

// func resetTable(db database.Querier, model any) error {
// 	tableName := db.Migrator().CurrentDatabase()
// 	if err := db.Migrator().DropTable(model); err != nil {
// 		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
// 	}
// 	if err := db.AutoMigrate(model); err != nil {
// 		return fmt.Errorf("failed to recreate table %s: %w", tableName, err)
// 	}
// 	return nil
// }
