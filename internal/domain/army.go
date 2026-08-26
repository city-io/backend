package domain

import "time"

// TroopType identifies a kind of troop and its stat profile.
type TroopType string

const (
	TroopTypeSoldier   TroopType = "soldier"
	TroopTypeArcher    TroopType = "archer"
	TroopTypeCavalry   TroopType = "cavalry"
	TroopTypeArtillery TroopType = "artillery"
)

// Army is a mobile group of troops on the map, owned by a player. Troops holds
// the count of each troop type. Order and battle details are live actor state;
// DestX/DestY remain persisted so movement can resume after restoration.
type Army struct {
	ArmyID        string              `json:"army_id"`
	Name          string              `json:"name"`
	Owner         string              `json:"owner"`
	X             int                 `json:"x"`
	Y             int                 `json:"y"`
	Troops        map[TroopType]int64 `json:"troops"`
	DestX         *int                `json:"dest_x"`
	DestY         *int                `json:"dest_y"`
	OrderID       *string             `json:"-"`
	OrderKind     ArmyOrderKind       `json:"-"`
	TargetArmyID  *string             `json:"-"`
	TargetCityID  *string             `json:"-"`
	BattleID      *string             `json:"-"`
	CaptureStart  *time.Time          `json:"-"`
	RemainingPath []Coordinates       `json:"-"`
	UpkeepCityID  *string             `json:"upkeep_city_id"`
	CreatedAt     time.Time           `json:"-"`
	UpdatedAt     time.Time           `json:"-"`
}

type ArmyOrderKind string

const (
	ArmyOrderMove    ArmyOrderKind = "move"
	ArmyOrderAttack  ArmyOrderKind = "attack"
	ArmyOrderConquer ArmyOrderKind = "conquer"
	ArmyOrderRetreat ArmyOrderKind = "retreat"
)

type BattleSide struct {
	UserIDs             []string
	ArmyIDs             []string
	MilitiaCityID       *string
	MilitiaCount        int64
	StartingTroops      map[TroopType]int64
	SurvivingTroops     map[TroopType]int64
	StartingMilitia     int64
	CumulativeLosses    BattleLossSummary
	LastRoundLosses     BattleLossSummary
	DefenseBonusPercent int
}

type BattleLossSummary struct {
	Troops    map[TroopType]int64
	Militia   int64
	Civilians int64
}

type Battle struct {
	BattleID        string
	X               int
	Y               int
	Attackers       BattleSide
	Defenders       BattleSide
	StartedAt       time.Time
	NextTick        time.Time
	CompletedRounds int
}
