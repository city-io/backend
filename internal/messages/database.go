package messages

import "cityio/internal/domain"

type GetEmptyCityBlockMessage struct {
	Size int
}
type GetEmptyCityBlockResponseMessage struct {
	X int
	Y int
}

// GetUserByIdentifierMessage looks a user up by email or username for login.
// Found is false when no row matched.
type GetUserByIdentifierMessage struct {
	Identifier string
}
type GetUserByIdentifierResponseMessage struct {
	User  domain.User
	Found bool
}

// GetMapMessage requests a full world snapshot read from the persistence layer.
type GetMapMessage struct{}
type GetMapResponseMessage struct {
	Cities    []domain.City
	Buildings []domain.Building
}
