package ws

import (
	"cityio/internal/models"
	"cityio/internal/services"

	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

func ProcessMessage(ctx context.Context, conn *websocket.Conn, messageType int, p []byte) error {
	var message models.WebSocketMessage
	if err := json.Unmarshal(p, &message); err != nil {
		log.Printf("Error decoding WebSocket message: %s", err)
		return err
	}

	switch message.Request {
	case "ping":
		conn.WriteMessage(messageType, []byte("pong"))
		return nil
	case "pong":
		return nil
	case "map":
		return getMapTiles(ctx, conn, &message)
	}

	return nil
}

func HandleWebSocket(response http.ResponseWriter, request *http.Request) {
	values := request.URL.Query()
	token := values.Get("token")
	if token == "" {
		log.Println("No token is provided")
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		return
	}

	claims, _, err := services.ValidateToken(token)
	if err != nil {
		log.Printf("Error parsing JWT: %s", err)
		http.Error(response, "Unauthorized", http.StatusUnauthorized)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// check origin for security
			return true
		},
	}

	conn, err := upgrader.Upgrade(response, request, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %s", err)
		http.Error(response, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	ctx := context.WithValue(request.Context(), "claims", claims)
	log.Printf("WebSocket connection established with %s", claims.Username)

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Println("Connection closed by client")
			} else {
				log.Printf("Error reading WebSocket message: %s", err)
			}
			break
		}

		err = ProcessMessage(ctx, conn, messageType, p)
		if err != nil {
			log.Printf("Error processing WebSocket message: %s", err)
			break
		}
	}
}
