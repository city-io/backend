package models

type WebSocketRequest struct {
	Req  int         `json:"req"`
	Data interface{} `json:"data"`
}

type UserClaims struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	UserId   string `json:"userId"`
}

type RegisterUserRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginUserRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type CityInput struct {
	Type  string `json:"type"`
	Owner string `json:"owner"`
	Name  string `json:"name"`
	Size  int    `json:"size"`
}

type MapTileRequest struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Radius int `json:"radius"`
}
