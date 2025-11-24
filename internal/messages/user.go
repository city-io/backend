package messages

import (
	"cityio/internal/models"

	"fmt"
)

type CreateUserMessage struct {
	User    models.User
	Restore bool
}
type CreateUserResponseMessage struct{}

type UpdateUserMessage struct {
	User models.User
}
type UpdateUserGoldMessage struct {
	Change int64
}
type UpdateUserFoodMessage struct {
	Change int64
}

type GetUserMessage struct{}
type GetUserResponseMessage struct {
	User models.User
}

type DeleteUserMessage struct {
	UserID string
}

// Errors
type UserNotFoundError struct {
	UserID string
}

func (e *UserNotFoundError) Error() string {
	return fmt.Sprintf("User not found: %s", e.UserID)
}

type InvalidPasswordError struct {
	Identifier string
}

func (e *InvalidPasswordError) Error() string {
	return fmt.Sprintf("Invalid password for user: %s", e.Identifier)
}

type InvalidTokenError struct{}

func (e *InvalidTokenError) Error() string {
	return "Invalid token"
}

type UserCreationError struct {
	UserID string
}

func (e *UserCreationError) Error() string {
	return fmt.Sprintf("Error creating user: %s", e.UserID)
}
