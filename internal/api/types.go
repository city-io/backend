package api

import "cityio/internal/domain"

// UserClaims is the set of identity claims carried in a JWT.
type UserClaims struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	UserId   string `json:"userId"`
}

// LoginUserRequest is the body of a login request.
type LoginUserRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// LoginUserResponse is returned on a successful login.
type LoginUserResponse struct {
	Token    string       `json:"token"`
	UserId   string       `json:"userId"`
	Email    string       `json:"email"`
	Username string       `json:"username"`
	Capital  *domain.City `json:"capital"`
}

// ValidateUserResponse is returned when validating a token.
type ValidateUserResponse struct {
	Username string       `json:"username"`
	Email    string       `json:"email"`
	UserId   string       `json:"userId"`
	Capital  *domain.City `json:"capital"`
}

// MapTileRequest requests the tiles around a coordinate.
type MapTileRequest struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Radius int `json:"radius"`
}
