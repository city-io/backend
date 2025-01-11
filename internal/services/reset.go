package services

import (
	"cityio/internal/constants"
	"cityio/internal/models"

	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Reset() {
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.MapTile{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.City{})
	db.Where("type = ?", "town").Delete(&models.City{})

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	occupied := make([][]bool, constants.MAP_SIZE)
	for i := range occupied {
		occupied[i] = make([]bool, constants.MAP_SIZE)
	}

	var users []models.User
	db.Find(&users)

	for _, user := range users {
		user.Balance = 1000
		db.Save(&user)

		src := rand.NewSource(time.Now().UnixNano())
		r := rand.New(src)

		startX := r.Intn(constants.MAP_SIZE - constants.CITY_SIZE)
		startY := r.Intn(constants.MAP_SIZE - constants.CITY_SIZE)
		cityId := uuid.New().String()
		result := db.Create(&models.City{
			CityId:     cityId,
			Type:       "city",
			Owner:      user.UserId,
			Name:       fmt.Sprintf("%s's City", user.Username),
			Population: 1000,
			StartX:     startX,
			StartY:     startY,
			Size:       constants.CITY_SIZE,
		})
		if result.Error != nil {
			log.Printf("Error creating city: %s", result.Error)
			panic(result.Error)
		} else {
			log.Printf("Created city %s for user %s", cityId, user.Username)
		}

		for i := 0; i < constants.CITY_SIZE; i++ {
			for j := 0; j < constants.CITY_SIZE; j++ {
				occupied[startX+i][startY+j] = true
			}
		}
	}

	cities := make([]models.City, 0)
	mapTiles := make([]models.MapTile, 0)
	for x := 0; x < constants.MAP_SIZE; x++ {
		for y := 0; y < constants.MAP_SIZE; y++ {
			if !occupied[x][y] {
				size := 0
				if r.Intn(100) < 2 {
					size = 3
				} else if r.Intn(100) < 8 {
					size = 2
				} else if r.Intn(100) < 15 {
					size = 1
				}
				if size > 0 && x+size < constants.MAP_SIZE && y+size < constants.MAP_SIZE {
					cityId := uuid.New().String()
					cities = append(cities, models.City{
						CityId:     cityId,
						Type:       "town",
						Owner:      "",
						Name:       fmt.Sprintf("Town %s", cityId),
						Population: constants.INITIAL_TOWN_POPULATION * size,
						StartX:     x,
						StartY:     y,
						Size:       size,
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

	tileBatchSize := 20000
	for i := 0; i < len(mapTiles); i += tileBatchSize {
		end := i + tileBatchSize
		if end > len(mapTiles) {
			end = len(mapTiles)
		}
		if result := db.Create(mapTiles[i:end]); result.Error != nil {
			log.Printf("Error creating map tiles: %s", result.Error)
		}
	}
}
