package domain

import "time"

// User is a player account.
type User struct {
	UserID   string `json:"userId"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	Gold     int64  `json:"gold"`
	Food     int64  `json:"food"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
