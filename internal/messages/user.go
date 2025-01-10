package messages

import (
	"fmt"
)

type GetUserMessage struct{}

// Errors
type UserNotFoundError struct {
	UserId string
}

func (e *UserNotFoundError) Error() string {
	return fmt.Sprintf("User not found: %s", e.UserId)
}
