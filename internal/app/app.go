package app

import (
	"cityio/internal/actors"
	"cityio/internal/api"
	"cityio/internal/constants"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/services"

	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var db = database.GetDb()
var system = actors.GetSystem()

func Start(reset bool) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if reset {
		Reset()
	}
	Init()
	api.Start()
}

func Init() {
	log.SetPrefix("[init]\t")
	managerPID := actors.GetManagerPID()
	initResponse, err := actors.Request[messages.InitPIDManagerResponseMessage](system.Root, managerPID, messages.InitPIDManagerMessage{})
	if err != nil {
		panic(err)
	}
	if initResponse.Error != nil {
		panic(initResponse.Error)
	}

	var users []models.User
	db.Find(&users)

	for _, user := range users {
		err := services.RestoreUser(user)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Spawned actors for %d users", len(users))

	var mapTiles []models.MapTile
	db.Find(&mapTiles)

	for _, mapTile := range mapTiles {
		err := services.RestoreMapTile(mapTile)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Spawned actors for %d map tiles", len(mapTiles))

	var cities []models.City
	db.Find(&cities)

	for _, city := range cities {
		err := services.RestoreCity(city)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Spawned actors for %d cities", len(cities))

	var armies []models.Army
	db.Find(&armies)

	for _, army := range armies {
		err := services.RestoreArmy(army)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Spawned actors for %d armies", len(armies))

	var buildings []models.Building
	db.Find(&buildings)

	for _, building := range buildings {
		err := services.RestoreBuilding(building)
		if err != nil {
			panic(err)
		}
	}
	log.Printf("Spawned actors for %d buildings", len(buildings))

	log.SetPrefix("[app]\t")
}

func Reset() {
	log.SetPrefix("[reset]\t")
	err := resetTable(db, &models.Army{})
	if err != nil {
		log.Fatalf("Error resetting Army table: %v", err)
	}

	err = resetTable(db, &models.MapTile{})
	if err != nil {
		log.Fatalf("Error resetting MapTile table: %v", err)
	}

	err = resetTable(db, &models.Building{})
	if err != nil {
		log.Fatalf("Error resetting Building table: %v", err)
	}

	err = resetTable(db, &models.City{})
	if err != nil {
		log.Fatalf("Error resetting City table: %v", err)
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
		cityId := uuid.New().String()
		result := db.Create(&models.City{
			CityId:        cityId,
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
			log.Printf("Error creating city: %s", result.Error)
			panic(result.Error)
		} else {
			log.Printf("Created city %s for user %s", cityId, user.Username)
		}

		result = db.Create(&models.Building{
			BuildingId: uuid.New().String(),
			CityId:     cityId,
			Type:       "city_center",
			Level:      1,
			X:          startX + int(math.Floor(float64(constants.CITY_SIZE)/2)),
			Y:          startY + int(math.Floor(float64(constants.CITY_SIZE)/2)),
		})

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
			if !occupied[x][y] {
				size := 0
				if r.Intn(1000) < 5 {
					size = 5
				} else if r.Intn(100) < 1 {
					size = 4
				} else if r.Intn(100) < 5 {
					size = 3
				} else if r.Intn(100) < 10 {
					size = 2
				}
				if size > 0 && x+size < constants.MAP_SIZE && y+size < constants.MAP_SIZE {
					cityId := uuid.New().String()
					cities = append(cities, models.City{
						CityId:        cityId,
						Type:          "town",
						Owner:         "",
						Name:          fmt.Sprintf("Town %s", cityId),
						Population:    constants.INITIAL_TOWN_POPULATION,
						PopulationCap: constants.GetBuildingPopulation(constants.BUILDING_TYPE_TOWN_CENTER, 1),
						StartX:        x,
						StartY:        y,
						Size:          size,
					})
					buildings = append(buildings, models.Building{
						BuildingId: uuid.New().String(),
						CityId:     cityId,
						Type:       "town_center",
						Level:      1,
						X:          x + int(math.Floor(float64(size)/2)),
						Y:          y + int(math.Floor(float64(size)/2)),
					})
					occupied[x][y] = true
					for i := 0; i < size-1; i++ {
						for j := 0; j < size-1; j++ {
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
		end := i + tileBatchSize
		if end > len(mapTiles) {
			end = len(mapTiles)
		}
		if result := db.Create(mapTiles[i:end]); result.Error != nil {
			log.Printf("Error creating map tiles: %s", result.Error)
		}
	}
	log.Printf("Created %d map tiles", len(mapTiles))

	cityBatchSize := 5000
	for i := 0; i < len(cities); i += cityBatchSize {
		end := i + cityBatchSize
		if end > len(cities) {
			end = len(cities)
		}
		if result := db.Create(cities[i:end]); result.Error != nil {
			log.Printf("Error creating cities: %s", result.Error)
		}
	}
	log.Printf("Created %d cities", len(cities))

	buildingBatchSize := 5000
	for i := 0; i < len(buildings); i += buildingBatchSize {
		end := i + buildingBatchSize
		if end > len(buildings) {
			end = len(buildings)
		}
		if result := db.Create(buildings[i:end]); result.Error != nil {
			log.Printf("Error creating buildings: %s", result.Error)
		}
	}
	log.Printf("Created %d buildings", len(buildings))

	log.Println("Reset complete!")
}

func resetTable(db *gorm.DB, model interface{}) error {
	tableName := db.Migrator().CurrentDatabase()
	if err := db.Migrator().DropTable(model); err != nil {
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}
	if err := db.AutoMigrate(model); err != nil {
		return fmt.Errorf("failed to recreate table %s: %w", tableName, err)
	}
	return nil
}
