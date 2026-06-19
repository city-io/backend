package messages

import (
	"fmt"

	"cityio/internal/domain"
)

type CreateUserMessage struct {
	User    domain.User
	Restore bool
}

// CreditUserMessage adds gold and/or food to a user in a single atomic update.
type CreditUserMessage struct {
	Gold int64
	Food int64
}

// DepositFoodMessage adds surplus food from a city to the user's pool. The
// amount is accumulated toward the user's rolling FoodIncomeRate.
type DepositFoodMessage struct {
	Amount int64
}

// RequestFoodFromPoolMessage is sent by a deficit city; the user grants up to
// the pool balance, withdrawing it from User.Food and accumulating it toward
// FoodUpkeepRate.
type RequestFoodFromPoolMessage struct {
	Amount int64
}

// RequestFoodFromPoolResponse reports how much of the request the pool could
// cover. Granted < Amount means the city is starving.
type RequestFoodFromPoolResponse struct {
	Granted int64
}

type CheckAndDeductGoldMessage struct {
	Amount int64
}

type GetUserMessage struct{}
type GetUserResponseMessage struct {
	User domain.User
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

type InsufficientGoldError struct {
	Missing int64
}

func (e *InsufficientGoldError) Error() string {
	return fmt.Sprintf("User has insufficient gold: %d", e.Missing)
}
