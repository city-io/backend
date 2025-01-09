package models

import (
	"time"
)

type Account struct {
	PlayerId  string    `json:"playerId" gorm:"column:player_id;primaryKey;size:50"`
	Username  string    `json:"username" gorm:"column:username;size:50;unique;not null"`
	Password  string    `json:"password" gorm:"column:password;size:64;not null"`
	Balance   float64   `json:"balance" gorm:"column:balance;default:2000000.0;not null;check:balance > 0"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}
