package models

import (
	"time"
)

type User struct {
	UserId   string `json:"userId" gorm:"column:user_id;primaryKey;size:36"`
	Email    string `json:"email" gorm:"column:email;size:100;unique;not null"`
	Username string `json:"username" gorm:"column:username;size:100;unique;not null"`
	Password string `json:"password" gorm:"column:password;size:64;not null"`
	Gold     int64  `json:"gold" gorm:"column:gold;not null;check:gold >= 0"`
	Food     int64  `json:"food" gorm:"column:food;not null;check:food >= 0"`

	Allies    []string  `json:"allies" gorm:"type:jsonb;serializer:json;not null;default:'[]'"`
	CreatedAt time.Time `json:"-" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"-" gorm:"column:updated_at;autoUpdateTime"`
}

type MapTile struct {
	X          int    `json:"x" gorm:"column:x;primaryKey;not null"`
	Y          int    `json:"y" gorm:"column:y;primaryKey;not null"`
	CityId     string `json:"cityId" gorm:"column:city_id;size:36;null"`
	BuildingId string `json:"buildingId" gorm:"column:building_id;size:36;null"`

	Armies []Army `json:"-" gorm:"foreignKey:TileX,TileY;references:X,Y"`
}

type City struct {
	CityId        string    `json:"cityId" gorm:"column:city_id;primaryKey;size:36"`
	Type          string    `json:"type" gorm:"column:type;size:100;not null"` // capital or town
	Owner         string    `json:"owner" gorm:"column:owner;size:36;null"`
	Name          string    `json:"name" gorm:"column:name;size:100;not null"`
	Population    float64   `json:"population" gorm:"column:population;not null;default:0;check:population >= 0"`
	PopulationCap float64   `json:"populationCap" gorm:"column:population_cap;not null;default:0;check:population_cap >= 0"`
	StartX        int       `json:"startX" gorm:"column:start_x;not null"`
	StartY        int       `json:"startY" gorm:"column:start_y;not null"`
	Size          int       `json:"size" gorm:"column:size;not null"`
	CreatedAt     time.Time `json:"-" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `json:"-" gorm:"column:updated_at;autoUpdateTime"`

	Buildings []Building `json:"-" gorm:"foreignKey:CityId;references:CityId"`
}

type Army struct {
	ArmyId string `json:"armyId" gorm:"column:army_id;primaryKey;size:36"`
	TileX  int    `json:"tileX" gorm:"column:tile_x;not null"`
	TileY  int    `json:"tileY" gorm:"column:tile_y;not null"`
	Owner  string `json:"owner" gorm:"column:owner;size:36;not null"`
	Size   int64  `json:"size" gorm:"column:size;not null;check:size > 0"`

	// march details
	FromX       int  `json:"fromX" gorm:"column:from_x;null"`
	FromY       int  `json:"fromY" gorm:"column:from_y;null"`
	ToX         int  `json:"toX" gorm:"column:to_x;null"`
	ToY         int  `json:"toY" gorm:"column:to_y;null"`
	MarchActive bool `json:"marchActive" gorm:"column:march_active;not null;default:false"`

	MapTile MapTile `json:"-" gorm:"foreignKey:TileX,TileY;references:X,Y"`
	User    User    `json:"-" gorm:"foreignKey:Owner;references:UserId"`
}

type Building struct {
	BuildingId      string    `json:"buildingId" gorm:"column:building_id;primaryKey;size:36"`
	CityId          string    `json:"cityId" gorm:"column:city_id;size:36;not null"`
	Type            string    `json:"type" gorm:"column:type;size:100;not null"`
	Level           int       `json:"level" gorm:"column:level;not null;default:1;check:level >= 0"`
	X               int       `json:"x" gorm:"column:x;uniqueIndex:compositeindex;not null"`
	Y               int       `json:"y" gorm:"column:y;uniqueIndex:compositeindex;not null"`
	ConstructionEnd time.Time `json:"constructionEnd" gorm:"column:construction_end;not null"`

	City City `json:"-"`
}

type Training struct {
	BarracksId string    `json:"barracksId" gorm:"column:barracks_id;primaryKey;size:36"`
	Size       int64     `json:"size" gorm:"column:size;not null;check:size > 0"`
	DeployTo   string    `json:"deployTo" gorm:"column:deploy_to;size:36;null"`
	End        time.Time `json:"end" gorm:"column:end;not null"`
}
