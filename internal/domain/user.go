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

	// FoodIncomeRate is the rolling food-per-hour deposited into the user's pool
	// by surplus cities. FoodUpkeepRate is the food-per-hour withdrawn by
	// importer cities. Sampled on the user actor's periodic tick.
	FoodIncomeRate int64 `json:"foodIncomeRate"`
	FoodUpkeepRate int64 `json:"foodUpkeepRate"`

	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}
