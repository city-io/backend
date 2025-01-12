package api

import (
	"cityio/internal/models"
	"cityio/internal/services"

	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

func GetMapTiles(response http.ResponseWriter, request *http.Request) {
	log.Println("Received GET /map/tiles")

	xStr := request.URL.Query().Get("x")
	yStr := request.URL.Query().Get("y")
	if xStr == "" || yStr == "" {
		response.WriteHeader(http.StatusBadRequest)
		return
	}
	x, err := strconv.Atoi(xStr)
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		return
	}
	y, err := strconv.Atoi(yStr)
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	radiusStr := request.URL.Query().Get("radius")
	if radiusStr == "" {
		radiusStr = "3"
	}
	radius, err := strconv.Atoi(radiusStr)

	var tiles []models.MapTileOutput
	for i := x - radius; i <= x+radius; i++ {
		for j := y - radius; j <= y+radius; j++ {
			tile, err := services.GetMapTile(i, j)
			if err != nil {
				log.Printf("Error getting map tile at x: %d, y: %d; %s", i, j, err.Error())
				continue
			}
			tiles = append(tiles, tile)
		}
	}
	log.Printf("Getting map tiles at x: %d, y: %d", x, y)

	json.NewEncoder(response).Encode(tiles)
}

func ResetMap(response http.ResponseWriter, request *http.Request) {
	log.Println("Received POST /map/reset")

	go services.Reset()
	response.WriteHeader(http.StatusAccepted)
}
