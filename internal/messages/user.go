package messages

import (
	"cityio/internal/models"

	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

type RegisterUserMessage struct {
	User    models.User
	Restore bool
}
type GetUserMessage struct{}
type AddUserArmyMessage struct {
	ArmyId  string
	ArmyPID *actor.PID
}
type DeleteUserMessage struct {
	UserId string
}

type RegisterUserResponseMessage struct {
	Error error
}
type GetUserResponseMessage struct {
	User models.User
}
type AddUserArmyResponseMessage struct {
	Error error
}
type DeleteUserResponseMessage struct {
	Error error
}

// Errors
type UserNotFoundError struct {
	UserId string
}

func (e *UserNotFoundError) Error() string {
	return fmt.Sprintf("User not found: %s", e.UserId)
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
	UserId string
}

func (e *UserCreationError) Error() string {
	return fmt.Sprintf("Error creating user: %s", e.UserId)
}
