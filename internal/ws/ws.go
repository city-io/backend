package ws

import (
	"log/slog"

	"github.com/gorilla/websocket"

	"cityio/internal/models"
)

var connections = make(map[string]*websocket.Conn)

func AddConnection(userId string, conn *websocket.Conn) {
	connections[userId] = conn
}

func Send(userId string, message int, data any) error {
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
			slog.Error("error broadcasting message", "error", err)
		}
	}
}
