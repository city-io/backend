package ws

// WebSocketRequest is an inbound message from a client socket.
type WebSocketRequest struct {
	Req  int `json:"req"`
	Data any `json:"data"`
}

// WebSocketResponse is an outbound message to a client socket.
type WebSocketResponse struct {
	Msg  int `json:"msg"`
	Data any `json:"data"`
}

// UserAccountOutput is the user account payload pushed over the socket.
type UserAccountOutput struct {
	Username string `json:"username"`
	Gold     int64  `json:"gold"`
	Food     int64  `json:"food"`
}
