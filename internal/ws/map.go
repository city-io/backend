package ws

import (
	"cityio/internal/constants"
	"cityio/internal/models"
	"cityio/internal/services"

	"context"
	"encoding/json"
	"errors"
	"log"
)

func getMapTiles(ctx context.Context, msg *models.WebSocketMessage) error {
	claims := ctx.Value("claims").(models.UserClaims)
	log.Printf("Fetching map tiles for %s", claims.Username)

	var data models.MapTileRequest
	if dataMap, ok := msg.Data.(map[string]interface{}); ok {
		dataBytes, err := json.Marshal(dataMap)
		if err != nil {
			log.Printf("Error marshalling data: %s", err)
			return err
		}

		if err := json.Unmarshal(dataBytes, &data); err != nil {
			log.Printf("Error unmarshalling data: %s", err)
			return err
		}
	} else {
		log.Println("Data is not in expected format.")
		return errors.New("Data is not in expected format.")
	}

	x, y := data.X, data.Y
	if x >= constants.MAP_SIZE || y >= constants.MAP_SIZE || x < 0 || y < 0 {
		log.Printf("Invalid coordinates: x: %d, y: %d", x, y)
		return nil
	}
	radius := data.Radius
	if radius == 0 {
		radius = 3
	}

	var tiles []models.MapTileOutput
	for i := x - radius; i <= x+radius; i++ {
		for j := y - radius; j <= y+radius; j++ {
			if i < 0 || j < 0 || i >= constants.MAP_SIZE || j >= constants.MAP_SIZE {
				continue
			}
			tile, err := services.GetMapTile(i, j)
			if err != nil {
				log.Printf("Error getting map tile at x: %d, y: %d; %s", i, j, err.Error())
				continue
			}
			tiles = append(tiles, tile)
		}
	}

	Send(claims.UserId, &tiles)

	return nil
}
