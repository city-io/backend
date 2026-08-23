package actors

import (
	"testing"

	"cityio/internal/domain"
	"cityio/internal/messages"
)

func TestBattleFractionalCarryEventuallyKillsWholeUnit(t *testing.T) {
	state := &battleActor{casualtyCarry: make(map[string]float64)}
	targets := []battleArmy{{id: "defender", army: domain.Army{Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 1}}}}

	var killed int64
	for tick := 0; tick < 16; tick++ {
		killed += state.casualties(targets, 10)["defender"][domain.TroopTypeSoldier]
		if killed > 0 {
			break
		}
	}
	if killed != 1 {
		t.Fatalf("casualties = %d, want one whole unit after fractional damage accumulates", killed)
	}
}

func TestBattleJoinAllowsDifferentUsersOnSameSide(t *testing.T) {
	state := &battleActor{Battle: domain.Battle{
		Attackers: domain.BattleSide{UserIDs: []string{"first"}, ArmyIDs: []string{"attacker"}},
		Defenders: domain.BattleSide{UserIDs: []string{"enemy"}, ArmyIDs: []string{"defender"}},
	}}

	if !state.join(messages.JoinBattleMessage{ArmyID: "ally", Owner: "second", OpposesArmyID: "defender"}) {
		t.Fatal("allied participant was rejected")
	}
	if !contains(state.Battle.Attackers.UserIDs, "second") || !contains(state.Battle.Attackers.ArmyIDs, "ally") {
		t.Fatalf("allied participant not added to attacker side: %+v", state.Battle.Attackers)
	}
}

func TestBattleCasualtiesAreCappedAtAvailableUnits(t *testing.T) {
	state := &battleActor{casualtyCarry: make(map[string]float64)}
	targets := []battleArmy{{id: "defender", army: domain.Army{Troops: map[domain.TroopType]int64{domain.TroopTypeArcher: 3}}}}

	got := state.casualties(targets, 1_000_000)["defender"][domain.TroopTypeArcher]
	if got != 3 {
		t.Fatalf("casualties = %d, want available count 3", got)
	}
}
