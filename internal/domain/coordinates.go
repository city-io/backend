// Package domain contains the core domain entities and value objects of the
// game, free of transport and persistence concerns.
package domain

// Coordinates is a position on the game map.
type Coordinates struct {
	X int `json:"x"`
	Y int `json:"y"`
}
