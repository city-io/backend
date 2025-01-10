package models

type UserClaims struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	UserId   string `json:"userId"`
}

type UserInput struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}
