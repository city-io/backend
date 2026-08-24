package actors

import (
	"errors"
	"testing"

	"cityio/internal/domain"
	"cityio/internal/messages"
)

func TestBattleFractionalCarryEventuallyKillsWholeUnit(t *testing.T) {
	state := &battleActor{casualtyCarry: make(map[string]float64)}
	targets := []battleArmy{{id: "defender", army: domain.Army{Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 1}}}}

	var killed int64
	for tick := 0; tick < 32; tick++ {
		casualties, _ := state.casualties(targets, nil, 0, 10)
		killed += casualties["defender"][domain.TroopTypeSoldier]
		if killed > 0 {
			break
		}
	}
	if killed != 1 {
		t.Fatalf("casualties = %d, want one whole unit after fractional damage accumulates", killed)
	}
}

func TestBattleCasualtyRateSpreadsLossAcrossRounds(t *testing.T) {
	state := &battleActor{casualtyCarry: make(map[string]float64)}
	targets := []battleArmy{{id: "defender", army: domain.Army{Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 1}}}}

	first, _ := state.casualties(targets, nil, 0, 150)
	second, _ := state.casualties(targets, nil, 0, 150)

	if first["defender"][domain.TroopTypeSoldier] != 0 || second["defender"][domain.TroopTypeSoldier] != 1 {
		t.Fatalf("casualties by round = %v then %v, want 0 then 1", first, second)
	}
}

func TestBattleJoinAllowsDifferentUsersOnSameSide(t *testing.T) {
	state := &battleActor{Battle: domain.Battle{
		Attackers: domain.BattleSide{UserIDs: []string{"first"}, ArmyIDs: []string{"attacker"}},
		Defenders: domain.BattleSide{UserIDs: []string{"enemy"}, ArmyIDs: []string{"defender"}},
	}}

	if !state.join(messages.JoinBattleMessage{Army: domain.Army{ArmyID: "ally", Owner: "second"}, OpposesArmyID: "defender"}) {
		t.Fatal("allied participant was rejected")
	}
	if !contains(state.Battle.Attackers.UserIDs, "second") || !contains(state.Battle.Attackers.ArmyIDs, "ally") {
		t.Fatalf("allied participant not added to attacker side: %+v", state.Battle.Attackers)
	}
}

func TestBattleJoinAllowsAlliesAgainstSettlementMilitia(t *testing.T) {
	cityID := "town"
	state := &battleActor{Battle: domain.Battle{
		Attackers: domain.BattleSide{UserIDs: []string{"first"}, ArmyIDs: []string{"attacker"}},
		Defenders: domain.BattleSide{MilitiaCityID: &cityID, MilitiaCount: 20},
	}}

	if !state.join(messages.JoinBattleMessage{Army: domain.Army{ArmyID: "ally", Owner: "second"}, OpposesMilitiaCityID: cityID}) {
		t.Fatal("allied participant was rejected")
	}
	if !contains(state.Battle.Attackers.UserIDs, "second") || !contains(state.Battle.Attackers.ArmyIDs, "ally") {
		t.Fatalf("allied participant not added against militia: %+v", state.Battle.Attackers)
	}
}

func TestBattleCasualtiesAreCappedAtAvailableUnits(t *testing.T) {
	state := &battleActor{casualtyCarry: make(map[string]float64)}
	targets := []battleArmy{{id: "defender", army: domain.Army{Troops: map[domain.TroopType]int64{domain.TroopTypeArcher: 3}}}}

	casualties, _ := state.casualties(targets, nil, 0, 1_000_000)
	got := casualties["defender"][domain.TroopTypeArcher]
	if got != 3 {
		t.Fatalf("casualties = %d, want available count 3", got)
	}
}

func TestBattleMilitiaCasualtiesAreCappedAtAvailableDefenders(t *testing.T) {
	state := &battleActor{casualtyCarry: make(map[string]float64)}
	cityID := "town"
	_, got := state.casualties(nil, &cityID, 3, 1_000_000)
	if got != 3 {
		t.Fatalf("militia casualties = %d, want available count 3", got)
	}
}

func TestSiegeAppliesCivilianCasualtiesButFieldBattleDoesNot(t *testing.T) {
	cityID := "town"
	applied := int64(0)
	cluster := &armyOperationTestCluster{request: func(kind, identity string, message any) (any, error) {
		if kind != "city" || identity != cityID {
			return nil, errors.New("unexpected request")
		}
		casualties, ok := message.(messages.ApplyCivilianCasualtiesMessage)
		if !ok {
			return nil, errors.New("unexpected message")
		}
		applied += casualties.Count
		return &messages.ApplyCivilianCasualtiesResponseMessage{Applied: casualties.Count}, nil
	}}
	state := &battleActor{
		baseActor:     baseActor{Cluster: cluster},
		Battle:        domain.Battle{BattleID: "battle"},
		casualtyCarry: make(map[string]float64),
	}

	if got := state.applySiegeCivilianCasualties(&domain.BattleSide{}, 10); got != 0 {
		t.Fatalf("field battle civilian casualties = %d, want 0", got)
	}
	if got := state.applySiegeCivilianCasualties(&domain.BattleSide{MilitiaCityID: &cityID}, 10); got != 1 {
		t.Fatalf("siege civilian casualties = %d, want 1", got)
	}
	if applied != 1 {
		t.Fatalf("applied civilian casualties = %d, want 1", applied)
	}
}

func TestSiegeCivilianCasualtiesScaleWithMilitaryLosses(t *testing.T) {
	cityID := "town"
	cluster := &armyOperationTestCluster{request: func(_, _ string, message any) (any, error) {
		casualties := message.(messages.ApplyCivilianCasualtiesMessage)
		return &messages.ApplyCivilianCasualtiesResponseMessage{Applied: casualties.Count}, nil
	}}
	state := &battleActor{
		baseActor:     baseActor{Cluster: cluster},
		Battle:        domain.Battle{BattleID: "battle"},
		casualtyCarry: make(map[string]float64),
	}
	side := &domain.BattleSide{MilitiaCityID: &cityID}

	if got := state.applySiegeCivilianCasualties(side, 1); got != 0 {
		t.Fatalf("fractional civilian casualties = %d, want 0", got)
	}
	if got := state.applySiegeCivilianCasualties(side, 20); got != 3 {
		t.Fatalf("large-round civilian casualties = %d, want 3", got)
	}
}

func TestMilitaryCasualtyCountIncludesArmiesAndMilitia(t *testing.T) {
	losses := map[string]map[domain.TroopType]int64{
		"first":  {domain.TroopTypeSoldier: 3, domain.TroopTypeArcher: 2},
		"second": {domain.TroopTypeCavalry: 4},
	}
	if got := militaryCasualtyCount(losses, 6); got != 15 {
		t.Fatalf("military casualty count = %d, want 15", got)
	}
}
