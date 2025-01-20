package ws

import (
	"cityio/internal/models"

	"context"
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

func ProcessMessage(ctx context.Context, conn *websocket.Conn, messageType int, p []byte) error {
	var message models.WebSocketMessage
	if err := json.Unmarshal(p, &message); err != nil {
		log.Printf("Error decoding WebSocket message: %s", err)
		return err
	}

	switch message.Request {
	case "map":
		return getMapTiles(ctx, conn, &message)
	}

	return nil
}
