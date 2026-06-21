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

	// FoodIncomeRate is the rolling food-per-second deposited into the user's
	// pool by surplus cities. FoodUpkeepRate is the food-per-second withdrawn
	// by importer cities. Sampled on the user actor's periodic tick.
	FoodIncomeRate float64 `json:"foodIncomeRate"`
	FoodUpkeepRate float64 `json:"foodUpkeepRate"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
