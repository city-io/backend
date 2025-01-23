package models

type LoginUserResponse struct {
	Token    string `json:"token"`
	UserId   string `json:"userId"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Capital  *City  `json:"capital"`
}

type ValidateUserResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	UserId   string `json:"userId"`
	Capital  *City  `json:"capital"`
}

type MapTileOutput struct {
	X        int                `json:"x"`
	Y        int                `json:"y"`
	City     *City              `json:"city"`
	Building *Building          `json:"building"`
	Armies   map[string][]*Army `json:"armies"`
}
