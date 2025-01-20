package ws

import (
	"cityio/internal/models"
	"cityio/internal/services"

	"context"
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

func getMapTiles(ctx context.Context, conn *websocket.Conn, msg *models.WebSocketMessage) error {
	claims := ctx.Value("claims").(models.UserClaims)
	log.Printf("[WS] Getting map tiles for %s", claims.Username)

	var data models.MapTileRequest
	if dataMap, ok := msg.Data.(map[string]interface{}); ok {
		dataBytes, err := json.Marshal(dataMap)
		if err != nil {
			log.Fatal(err)
		}

		if err := json.Unmarshal(dataBytes, &data); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("Data is not in expected format.")
	}

	x, y := data.X, data.Y
	radius := data.Radius
	if radius == 0 {
		radius = 3
	}

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

	conn.WriteJSON(&tiles)

	return nil
}
