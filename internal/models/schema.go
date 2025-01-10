package models

import (
	"time"
)

type User struct {
	UserId    string    `json:"userId" gorm:"column:user_id;primaryKey;size:36"`
	Email     string    `json:"email" gorm:"column:email;size:100;unique;not null"`
	Username  string    `json:"username" gorm:"column:username;size:100;unique;not null"`
	Password  string    `json:"password" gorm:"column:password;size:64;not null"`
	Balance   float64   `json:"balance" gorm:"column:balance;default:2000000.0;not null;check:balance > 0"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`

	City City `json:"city" gorm:"foreignKey:Owner;references:UserId"`
}

type MapTile struct {
	X      int    `json:"x" gorm:"column:x;primaryKey;not null"`
	Y      int    `json:"y" gorm:"column:y;primaryKey;not null"`
	CityId string `json:"cityId" gorm:"column:city_id;size:36;null"`

	Armies []Army `json:"armies" gorm:"foreignKey:TileX,TileY;references:X,Y"`
}

type City struct {
	CityId     string    `json:"cityId" gorm:"column:city_id;primaryKey;size:36"`
	Type       string    `json:"type" gorm:"column:type;size:100;not null"` // city or town
	Owner      string    `json:"owner" gorm:"column:owner;size:36;null"`
	Name       string    `json:"name" gorm:"column:name;size:100;not null"`
	Population int       `json:"population" gorm:"column:population;not null;check:population >= 0"`
	StartX     int       `json:"startX" gorm:"column:start_x;not null"`
	StartY     int       `json:"startY" gorm:"column:start_y;not null"`
	Size       int       `json:"size" gorm:"column:size;not null;default:4"`
	CreatedAt  time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

type Army struct {
	ArmyId    string    `json:"armyId" gorm:"column:army_id;primaryKey;size:36"`
	TileX     int       `json:"tileX" gorm:"column:tile_x;not null"` // Matches MapTile.X
	TileY     int       `json:"tileY" gorm:"column:tile_y;not null"` // Matches MapTile.Y
	Owner     string    `json:"owner" gorm:"column:owner;size:36;not null"`
	Size      int       `json:"size" gorm:"column:size;not null;check:size > 0"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`

	MapTile MapTile `json:"mapTile" gorm:"foreignKey:TileX,TileY;references:X,Y"`
	User    User    `json:"user" gorm:"foreignKey:Owner;references:UserId"`
}
