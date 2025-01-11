package services

import (
	"cityio/internal/models"

	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Reset() {
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.MapTile{})
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.City{})

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	mapSize := 1024
	occupied := make([][]bool, mapSize)
	for i := range occupied {
		occupied[i] = make([]bool, mapSize)
	}

	cities := make([]models.City, 0)
	mapTiles := make([]models.MapTile, 0)
	for x := 0; x < mapSize; x++ {
		for y := 0; y < mapSize; y++ {
			if !occupied[x][y] {
				size := 0
				if r.Intn(100) < 2 {
					size = 3
				} else if r.Intn(100) < 8 {
					size = 2
				} else if r.Intn(100) < 15 {
					size = 1
				}
				if size > 0 && x+size < mapSize && y+size < mapSize {
					cities = append(cities, models.City{
						CityId: uuid.New().String(),
						Type:   "city",
						Owner:  "",
						Name:   "Town",
						StartX: x,
						StartY: y,
						Size:   size,
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
		} else {
			log.Printf("Created %d cities in batch", result.RowsAffected)
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
		} else {
			log.Printf("Created %d map tiles in batch", result.RowsAffected)
		}
	}
}
