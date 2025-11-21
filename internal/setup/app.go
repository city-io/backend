// Package setup initializes the application state by restoring actors from the database.
package setup

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"cityio/internal/constants"
	"cityio/internal/models"
	"cityio/internal/ports"
	"cityio/internal/services"
)

func Run(log ports.Logger, cl ports.ClusterProvider) {
	reset(log, cl.DB())
	log = log.With("phase", "init")

	var users []models.User
	cl.DB().Find(&users)

	for _, user := range users {
		err := services.RestoreUser(cl, user)
		if err != nil {
			panic(err)
		}
	}
	log.Info("Spawned user actors", "count", len(users))

	// var mapTiles []models.MapTile
	// cl.DB().Find(&mapTiles)

	// for _, mapTile := range mapTiles {
	// 	err := services.RestoreMapTile(mapTile)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }
	// log.Printf("Spawned actors for %d map tiles", len(mapTiles))

	var cities []models.City
	cl.DB().Find(&cities)

	for _, city := range cities {
		err := services.RestoreCity(cl, city)
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

func reset(log ports.Logger, db *gorm.DB) error {
	log = log.With("phase", "reset")
	err := resetTable(db, &models.Army{})
	if err != nil {
		log.Error("Error resetting Army table", "error", err)
		return err
	}

	err = resetTable(db, &models.MapTile{})
	if err != nil {
		log.Error("Error resetting MapTile table", "error", err)
		return err
	}

	err = resetTable(db, &models.Building{})
	if err != nil {
		log.Error("Error resetting Building table", "error", err)
		return err
	}

	err = resetTable(db, &models.City{})
	if err != nil {
		log.Error("Error resetting City table", "error", err)
		return err
	}

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	occupied := make([][]bool, constants.MAP_SIZE)
	for i := range occupied {
		occupied[i] = make([]bool, constants.MAP_SIZE)
	}

	var users []models.User
	db.Find(&users)

	for _, user := range users {
		user.Gold = constants.INITIAL_PLAYER_GOLD
		user.Food = constants.INITIAL_PLAYER_FOOD
		db.Save(&user)

		src := rand.NewSource(time.Now().UnixNano())
		r := rand.New(src)
		startX := r.Intn(constants.MAP_SIZE - constants.CITY_SIZE)
		startY := r.Intn(constants.MAP_SIZE - constants.CITY_SIZE)

		cityID := uuid.New().String()
		result := db.Create(&models.City{
			CityId:        cityID,
			Type:          "capital",
			Owner:         user.UserId,
			Name:          fmt.Sprintf("%s's City", user.Username),
			Population:    constants.INITIAL_PLAYER_CITY_POPULATION,
			PopulationCap: constants.GetBuildingPopulation(constants.BUILDING_TYPE_CITY_CENTER, 1),
			StartX:        startX,
			StartY:        startY,
			Size:          constants.CITY_SIZE,
		})
		if result.Error != nil {
			log.Error("Error creating city in db", "error", result.Error)
			return result.Error
		} else {
			log.Debug("Created city in db", "cityId", cityID, "user", user.Username, "x", startX, "y", startY)
		}

		result = db.Create(&models.Building{
			BuildingId: uuid.New().String(),
			CityId:     cityID,
			Type:       "city_center",
			Level:      1,
			X:          startX + int(math.Floor(float64(constants.CITY_SIZE)/2)),
			Y:          startY + int(math.Floor(float64(constants.CITY_SIZE)/2)),
		})
		if result.Error != nil {
			log.Error("Error creating building in db", "error", result.Error)
			return result.Error
		}

		for i := 0; i < constants.CITY_SIZE; i++ {
			for j := 0; j < constants.CITY_SIZE; j++ {
				occupied[startX+i][startY+j] = true
			}
		}
	}

	cities := make([]models.City, 0)
	buildings := make([]models.Building, 0)
	mapTiles := make([]models.MapTile, 0)
	for x := 0; x < constants.MAP_SIZE; x++ {
		for y := 0; y < constants.MAP_SIZE; y++ {
			open := true
			// TODO: optimize random city placement
			for i := -1; i < 6; i++ {
				for j := -1; j < 6; j++ {
					if x+i < 0 || y+j < 0 || x+i >= constants.MAP_SIZE || y+j >= constants.MAP_SIZE || occupied[x+i][y+j] {
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
				if size > 0 && x+size < constants.MAP_SIZE && y+size < constants.MAP_SIZE {
					cityID := uuid.New().String()
					cities = append(cities, models.City{
						CityId:        cityID,
						Type:          "town",
						Owner:         "",
						Name:          fmt.Sprintf("Town %s", cityID),
						Population:    constants.INITIAL_TOWN_POPULATION,
						PopulationCap: constants.GetBuildingPopulation(constants.BUILDING_TYPE_TOWN_CENTER, 1),
						StartX:        x,
						StartY:        y,
						Size:          size,
					})
					buildings = append(buildings, models.Building{
						BuildingId: uuid.New().String(),
						CityId:     cityID,
						Type:       "town_center",
						Level:      1,
						X:          x + int(math.Floor(float64(size)/2)),
						Y:          y + int(math.Floor(float64(size)/2)),
					})
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

	tileBatchSize := 15000
	for i := 0; i < len(mapTiles); i += tileBatchSize {
		end := min(i+tileBatchSize, len(mapTiles))
		if result := db.Create(mapTiles[i:end]); result.Error != nil {
			log.Error("Error creating map tiles", "error", result.Error)
			return result.Error
		}
	}
	log.Debug("Created map tiles", "count", len(mapTiles))

	cityBatchSize := 5000
	for i := 0; i < len(cities); i += cityBatchSize {
		end := min(i+cityBatchSize, len(cities))
		if result := db.Create(cities[i:end]); result.Error != nil {
			log.Error("Error creating cities", "error", result.Error)
			return result.Error
		}
	}
	log.Debug("Created cities", "count", len(cities))

	buildingBatchSize := 5000
	for i := 0; i < len(buildings); i += buildingBatchSize {
		end := min(i+buildingBatchSize, len(buildings))
		if result := db.Create(buildings[i:end]); result.Error != nil {
			log.Error("Error creating buildings", "error", result.Error)
			return result.Error
		}
	}
	log.Debug("Created buildings", "count", len(buildings))
	log.Debug("Reset complete!")
	return nil
}

func resetTable(db *gorm.DB, model any) error {
	tableName := db.Migrator().CurrentDatabase()
	if err := db.Migrator().DropTable(model); err != nil {
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}
	if err := db.AutoMigrate(model); err != nil {
		return fmt.Errorf("failed to recreate table %s: %w", tableName, err)
	}
	return nil
}
