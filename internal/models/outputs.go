package models

type LoginUserResponse struct {
	Token    string `json:"token"`
	UserId   string `json:"userId"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type MapTileOutput struct {
	X    int   `json:"x"`
	Y    int   `json:"y"`
	City *City `json:"city"`
}
