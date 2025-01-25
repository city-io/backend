package ws

import (
	"cityio/internal/models"

	"log"

	"github.com/gorilla/websocket"
)

var connections = make(map[string]*websocket.Conn)

func AddConnection(userId string, conn *websocket.Conn) {
	connections[userId] = conn
}

func Send(userId string, message int, data interface{}) error {
	conn, ok := connections[userId]
	if !ok {
		return nil
	}

	return conn.WriteJSON(&models.WebSocketResponse{
		Msg:  message,
		Data: data,
	})
}

func Broadcast(message interface{}) {
	for _, conn := range connections {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("Error broadcasting message: %s", err)
		}
	}
}
