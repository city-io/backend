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
// the count of each troop type. MarchID and DestX/DestY describe its active
// movement order; all three are nil while idle. MarchID is live actor state,
// while the destination is persisted so movement can resume after restoration.
// UpkeepCityID caches the owned city currently bearing this army's food upkeep.
type Army struct {
	ArmyID       string              `json:"army_id"`
	Owner        string              `json:"owner"`
	X            int                 `json:"x"`
	Y            int                 `json:"y"`
	Troops       map[TroopType]int64 `json:"troops"`
	DestX        *int                `json:"dest_x"`
	DestY        *int                `json:"dest_y"`
	MarchID      *string             `json:"-"`
	UpkeepCityID *string             `json:"upkeep_city_id"`
	CreatedAt    time.Time           `json:"-"`
	UpdatedAt    time.Time           `json:"-"`
}
