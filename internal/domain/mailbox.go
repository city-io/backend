package domain

import "time"

type BattleReportRole string

const (
	BattleReportRoleAttacker BattleReportRole = "attacker"
	BattleReportRoleDefender BattleReportRole = "defender"
)

type BattleReportOutcome string

const (
	BattleReportOutcomeVictory BattleReportOutcome = "victory"
	BattleReportOutcomeDefeat  BattleReportOutcome = "defeat"
	BattleReportOutcomeDraw    BattleReportOutcome = "draw"
)

type BattleReportEngagement string

const (
	BattleReportEngagementField BattleReportEngagement = "field_battle"
	BattleReportEngagementSiege BattleReportEngagement = "settlement_siege"
)

type BattleReportResolution string

const (
	BattleReportResolutionElimination       BattleReportResolution = "elimination"
	BattleReportResolutionRetreat           BattleReportResolution = "retreat"
	BattleReportResolutionMutualDestruction BattleReportResolution = "mutual_destruction"
)

type BattleReportArmy struct {
	ArmyID          string              `json:"army_id"`
	OwnerID         string              `json:"owner_id"`
	StartingTroops  map[TroopType]int64 `json:"starting_troops"`
	SurvivingTroops map[TroopType]int64 `json:"surviving_troops"`
	Retreated       bool                `json:"retreated"`
	Destroyed       bool                `json:"destroyed"`
}

type BattleReportCommander struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

type BattleReportSettlement struct {
	CityID             string   `json:"city_id"`
	Name               string   `json:"name"`
	Type               CityType `json:"type"`
	OwnerID            *string  `json:"owner_id,omitempty"`
	StartingPopulation float64  `json:"starting_population"`
	EndingPopulation   float64  `json:"ending_population"`
}

type BattleReportSide struct {
	UserIDs          []string                `json:"user_ids"`
	Commanders       []BattleReportCommander `json:"commanders"`
	Armies           []BattleReportArmy      `json:"armies"`
	MilitiaCityID    *string                 `json:"militia_city_id,omitempty"`
	StartingMilitia  int64                   `json:"starting_militia"`
	SurvivingMilitia int64                   `json:"surviving_militia"`
	Settlement       *BattleReportSettlement `json:"settlement,omitempty"`
}

type BattleReportLoss struct {
	ArmyID        *string             `json:"army_id,omitempty"`
	MilitiaCityID *string             `json:"militia_city_id,omitempty"`
	Troops        map[TroopType]int64 `json:"troops,omitempty"`
	Militia       int64               `json:"militia,omitempty"`
}

type BattleReportRound struct {
	Number         int                `json:"number"`
	OccurredAt     time.Time          `json:"occurred_at"`
	AttackerPower  float64            `json:"attacker_power"`
	DefenderPower  float64            `json:"defender_power"`
	AttackerLosses []BattleReportLoss `json:"attacker_losses"`
	DefenderLosses []BattleReportLoss `json:"defender_losses"`
}

type BattleReport struct {
	BattleID   string                 `json:"battle_id"`
	X          int                    `json:"x"`
	Y          int                    `json:"y"`
	Role       BattleReportRole       `json:"role"`
	Outcome    BattleReportOutcome    `json:"outcome"`
	Engagement BattleReportEngagement `json:"engagement"`
	Resolution BattleReportResolution `json:"resolution"`
	Attackers  BattleReportSide       `json:"attackers"`
	Defenders  BattleReportSide       `json:"defenders"`
	Rounds     []BattleReportRound    `json:"rounds"`
	StartedAt  time.Time              `json:"started_at"`
	EndedAt    time.Time              `json:"ended_at"`
}

type MailboxMessage struct {
	MailboxMessageID string
	RecipientID      string
	CreatedAt        time.Time
	ReadAt           NullTime
	BattleReport     *BattleReport
}
