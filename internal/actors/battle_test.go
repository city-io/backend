package actors

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cityio/internal/constants"
	"cityio/internal/contracts"
	"cityio/internal/domain"
	"cityio/internal/messages"
)

type battleDefenseTestStore struct {
	contracts.Store
	buildings []domain.Building
}

func (s *battleDefenseTestStore) GetBuildingsByCity(context.Context, string) ([]domain.Building, error) {
	return s.buildings, nil
}

func TestBattleRoundsUseThreeSecondCadence(t *testing.T) {
	if constants.BattleTickInterval != 3*time.Second {
		t.Fatalf("battle tick interval = %s, want 3s", constants.BattleTickInterval)
	}
}

func TestBattleCasualtiesVaryAcrossBattleSeeds(t *testing.T) {
	targets := []battleArmy{{id: "defender", army: domain.Army{Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 15}}}}
	outcomes := make(map[int64]struct{})
	for seed := 0; seed < 128; seed++ {
		state := &battleActor{Battle: domain.Battle{BattleID: fmt.Sprintf("battle-%d", seed)}}
		casualties, _ := state.casualties(targets, nil, 0, 150, 0)
		outcomes[casualties["defender"][domain.TroopTypeSoldier]] = struct{}{}
	}
	if len(outcomes) < 2 {
		t.Fatalf("casualty outcomes = %v, want variation across battle seeds", outcomes)
	}
}

func TestBattleCasualtiesAreReproducibleForBattleSeed(t *testing.T) {
	targets := []battleArmy{{id: "defender", army: domain.Army{Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 15}}}}
	first := &battleActor{Battle: domain.Battle{BattleID: "same-battle"}}
	second := &battleActor{Battle: domain.Battle{BattleID: "same-battle"}}
	for round := 0; round < 20; round++ {
		firstLosses, _ := first.casualties(targets, nil, 0, 150, 0)
		secondLosses, _ := second.casualties(targets, nil, 0, 150, 0)
		firstCount := firstLosses["defender"][domain.TroopTypeSoldier]
		secondCount := secondLosses["defender"][domain.TroopTypeSoldier]
		if firstCount != secondCount {
			t.Fatalf("round %d casualties = %d and %d for the same battle seed", round+1, firstCount, secondCount)
		}
	}
}

func TestBattleRandomCasualtiesPreserveExpectedLosses(t *testing.T) {
	targets := []battleArmy{{id: "defender", army: domain.Army{Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 15}}}}
	var total int64
	const samples = 4096
	for seed := 0; seed < samples; seed++ {
		state := &battleActor{Battle: domain.Battle{BattleID: fmt.Sprintf("battle-%d", seed)}}
		casualties, _ := state.casualties(targets, nil, 0, 100, 0)
		total += casualties["defender"][domain.TroopTypeSoldier]
	}
	mean := float64(total) / samples
	if mean < 0.60 || mean > 0.73 {
		t.Fatalf("mean casualties = %f, want approximately 0.667", mean)
	}
}

func TestBattleDefenseBonusReducesExpectedLosses(t *testing.T) {
	targets := []battleArmy{{id: "defender", army: domain.Army{Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 15}}}}
	var total int64
	const samples = 4096
	for seed := 0; seed < samples; seed++ {
		state := &battleActor{Battle: domain.Battle{BattleID: fmt.Sprintf("defended-battle-%d", seed)}}
		casualties, _ := state.casualties(targets, nil, 0, 150, 50)
		total += casualties["defender"][domain.TroopTypeSoldier]
	}
	mean := float64(total) / samples
	if mean < 0.60 || mean > 0.73 {
		t.Fatalf("mean defended casualties = %f, want approximately 0.667", mean)
	}
}

func TestBattleSettlementAndFortDefenseBonusesAdd(t *testing.T) {
	cityID := "town"
	state := &battleActor{baseActor: baseActor{Store: &battleDefenseTestStore{buildings: []domain.Building{{
		Type: string(domain.BuildingTypeTownCenter), Level: 3,
	}}}, Cluster: &armyOperationTestCluster{}}}
	fort := &domain.Building{Owner: "owner", Type: string(domain.BuildingTypeFort), Level: 3}
	got := state.defenseBonusPercent(domain.BattleSide{UserIDs: []string{"owner"}, MilitiaCityID: &cityID}, fort, nil)
	if got != 35 {
		t.Fatalf("combined defense bonus = %d, want 35", got)
	}
}

func TestBattleCasualtiesScaleWithForceSize(t *testing.T) {
	meanLossesForEqualBattle := func(count int64) float64 {
		var total int64
		const samples = 2048
		for seed := 0; seed < samples; seed++ {
			state := &battleActor{Battle: domain.Battle{BattleID: fmt.Sprintf("battle-%d-%d", count, seed)}}
			army := battleArmy{id: "army", army: domain.Army{Troops: map[domain.TroopType]int64{
				domain.TroopTypeSoldier: count,
			}}}
			casualties, _ := state.casualties([]battleArmy{army}, nil, 0, attackPower([]battleArmy{army}, 0), 0)
			total += casualties[army.id][domain.TroopTypeSoldier]
		}
		return float64(total) / samples
	}

	small := meanLossesForEqualBattle(15)
	large := meanLossesForEqualBattle(150)
	if small < 0.90 || small > 1.10 || large < 9.50 || large > 10.50 {
		t.Fatalf("mean equal-battle casualties: 15v15 = %f, 150v150 = %f; want approximately 1 and 10", small, large)
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
	state := &battleActor{Battle: domain.Battle{BattleID: "capped-army-casualties"}}
	targets := []battleArmy{{id: "defender", army: domain.Army{Troops: map[domain.TroopType]int64{domain.TroopTypeArcher: 3}}}}

	casualties, _ := state.casualties(targets, nil, 0, 1_000_000, 0)
	got := casualties["defender"][domain.TroopTypeArcher]
	if got != 3 {
		t.Fatalf("casualties = %d, want available count 3", got)
	}
}

func TestBattleMilitiaCasualtiesAreCappedAtAvailableDefenders(t *testing.T) {
	state := &battleActor{Battle: domain.Battle{BattleID: "capped-militia-casualties"}}
	cityID := "town"
	_, got := state.casualties(nil, &cityID, 3, 1_000_000, 0)
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
		baseActor:             baseActor{Cluster: cluster},
		Battle:                domain.Battle{BattleID: "battle"},
		civilianCasualtyCarry: make(map[string]float64),
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
		baseActor:             baseActor{Cluster: cluster},
		Battle:                domain.Battle{BattleID: "battle"},
		civilianCasualtyCarry: make(map[string]float64),
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

func TestLiveBattleSummaryPreservesStrengthAndLosses(t *testing.T) {
	report := domain.BattleReportSide{
		StartingMilitia: 10, SurvivingMilitia: 7,
		Armies: []domain.BattleReportArmy{{
			StartingTroops:  map[domain.TroopType]int64{domain.TroopTypeSoldier: 12, domain.TroopTypeArcher: 8},
			SurvivingTroops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 9, domain.TroopTypeArcher: 6},
		}},
		Settlement: &domain.BattleReportSettlement{CivilianCasualties: 2},
	}
	last := []domain.BattleReportLoss{{Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 2}, Militia: 1}}
	var side domain.BattleSide

	syncLiveBattleSide(&side, report, last, 1)

	if side.StartingTroops[domain.TroopTypeSoldier] != 12 || side.SurvivingTroops[domain.TroopTypeSoldier] != 9 {
		t.Fatalf("live strength = %+v / %+v", side.StartingTroops, side.SurvivingTroops)
	}
	if side.CumulativeLosses.Troops[domain.TroopTypeSoldier] != 3 || side.CumulativeLosses.Troops[domain.TroopTypeArcher] != 2 || side.CumulativeLosses.Militia != 3 || side.CumulativeLosses.Civilians != 2 {
		t.Fatalf("cumulative losses = %+v", side.CumulativeLosses)
	}
	if side.LastRoundLosses.Troops[domain.TroopTypeSoldier] != 2 || side.LastRoundLosses.Militia != 1 || side.LastRoundLosses.Civilians != 1 {
		t.Fatalf("last round losses = %+v", side.LastRoundLosses)
	}
}
