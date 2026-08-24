package mapping

import (
	"testing"
	"time"

	"cityio/internal/domain"
	entityv1 "cityio/internal/gen/cityio/entity/v1"
)

func TestMapTilesToProtoBuildsCoordinateKeyedEntityGraph(t *testing.T) {
	grid := domain.TerrainGrid{
		Width:  3,
		Height: 2,
		Tiles: []domain.TerrainType{
			domain.TerrainTypeGrassland,
			domain.TerrainTypePlains,
			domain.TerrainTypeWater,
			domain.TerrainTypeForest,
			domain.TerrainTypeHills,
			domain.TerrainTypeMountains,
		},
	}
	cities := []domain.City{{CityID: "city-1", StartX: 0, StartY: 0, Size: 2}}
	buildings := []domain.Building{{BuildingID: "building-1", X: 1, Y: 0}}
	armies := []domain.Army{
		{ArmyID: "army-1", X: 1, Y: 1},
		{ArmyID: "army-2", X: 1, Y: 1},
	}

	ids, tiles := MapTilesToProto(grid, cities, buildings, armies)
	if len(ids) != 6 || len(tiles) != 6 {
		t.Fatalf("got %d ids and %d tiles, want 6 of each", len(ids), len(tiles))
	}
	for idx, id := range ids {
		wantX, wantY := int32(idx%grid.Width), int32(idx/grid.Width)
		if id.GetX() != wantX || id.GetY() != wantY {
			t.Fatalf("id %d = (%d,%d), want (%d,%d)", idx, id.GetX(), id.GetY(), wantX, wantY)
		}
		if tiles[idx].GetTileId() != id {
			t.Fatalf("tile %d does not share its root ID", idx)
		}
	}

	occupied := tiles[1]
	if occupied.GetCityId().GetValue() != "city-1" {
		t.Fatalf("city id = %q, want city-1", occupied.GetCityId().GetValue())
	}
	if occupied.GetBuildingId().GetValue() != "building-1" {
		t.Fatalf("building id = %q, want building-1", occupied.GetBuildingId().GetValue())
	}
	stacked := tiles[4]
	if len(stacked.GetArmyIds()) != 2 {
		t.Fatalf("army ids = %d, want 2", len(stacked.GetArmyIds()))
	}
	if tiles[2].GetCityId() != nil || tiles[2].GetBuildingId() != nil || len(tiles[2].GetArmyIds()) != 0 {
		t.Fatal("unoccupied tile contains occupancy references")
	}
}

func TestMailboxMessageToProtoPreservesDetailedBattleReport(t *testing.T) {
	started := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	ended := started.Add(8 * time.Second)
	cityID, ownerID := "town", "defender"
	message := domain.MailboxMessage{
		MailboxMessageID: "message", RecipientID: "attacker", CreatedAt: ended,
		BattleReport: &domain.BattleReport{
			BattleID: "battle", X: 4, Y: 5, Role: domain.BattleReportRoleAttacker,
			Outcome: domain.BattleReportOutcomeVictory, Engagement: domain.BattleReportEngagementSiege,
			Resolution: domain.BattleReportResolutionElimination, StartedAt: started, EndedAt: ended,
			Attackers: domain.BattleReportSide{
				UserIDs: []string{"attacker"}, Commanders: []domain.BattleReportCommander{{UserID: "attacker", Username: "Alice"}},
				Armies: []domain.BattleReportArmy{{ArmyID: "army", OwnerID: "attacker", StartingTroops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 10}, SurvivingTroops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 7}}},
			},
			Defenders: domain.BattleReportSide{
				UserIDs: []string{"defender"}, MilitiaCityID: &cityID, StartingMilitia: 12, SurvivingMilitia: 0,
				Settlement: &domain.BattleReportSettlement{CityID: cityID, Name: "Ashford", Type: domain.CityTypeTown, OwnerID: &ownerID, StartingPopulation: 80, EndingPopulation: 68, CivilianCasualties: 4},
			},
			Rounds: []domain.BattleReportRound{{Number: 1, OccurredAt: ended, AttackerPower: 100, DefenderPower: 120, DefenderLosses: []domain.BattleReportLoss{{MilitiaCityID: &cityID, Militia: 12}}, DefenderCivilianCasualties: 4}},
		},
	}

	mapped := MailboxMessageToProto(message)
	report := mapped.GetBattleReport()
	if mapped.GetMailboxMessageId().GetValue() != "message" || report.GetBattleId().GetValue() != "battle" || report.GetTileId().GetX() != 4 || report.GetTileId().GetY() != 5 {
		t.Fatalf("report identity was not preserved: %+v", mapped)
	}
	if report.GetAttackers().GetCommanders()[0].GetUsername() != "Alice" || report.GetAttackers().GetArmies()[0].GetStartingTroops()[0].GetCount() != 10 {
		t.Fatalf("attacker detail was not preserved: %+v", report.GetAttackers())
	}
	if !report.GetDefenders().GetStrengthVisible() || report.GetDefenders().GetSettlement().GetName() != "Ashford" || report.GetDefenders().GetStartingMilitia() != 12 || report.GetRounds()[0].GetDefenderLosses()[0].GetMilitia() != 12 || report.GetDefenders().GetSettlement().GetCivilianCasualties() != 4 || report.GetRounds()[0].GetDefenderCivilianCasualties() != 4 {
		t.Fatalf("siege or round detail was not preserved: %+v", report)
	}
}

func TestBattleProjectionHidesOpposingMilitiaStrength(t *testing.T) {
	cityID := "town"
	battle := domain.Battle{
		Attackers: domain.BattleSide{UserIDs: []string{"attacker"}},
		Defenders: domain.BattleSide{UserIDs: []string{"defender"}, MilitiaCityID: &cityID, MilitiaCount: 25},
	}

	mapped := BattleToProto(battle, "attacker")

	if !mapped.GetAttackers().GetStrengthVisible() {
		t.Fatal("friendly battle strength was hidden")
	}
	if mapped.GetDefenders().GetStrengthVisible() || mapped.GetDefenders().GetMilitiaCount() != 0 {
		t.Fatalf("opposing militia strength was exposed: %+v", mapped.GetDefenders())
	}
}

func TestDefeatReportHidesOpposingCounts(t *testing.T) {
	cityID := "town"
	message := domain.MailboxMessage{BattleReport: &domain.BattleReport{
		Role: domain.BattleReportRoleAttacker, Outcome: domain.BattleReportOutcomeDefeat,
		Attackers: domain.BattleReportSide{UserIDs: []string{"attacker"}, Armies: []domain.BattleReportArmy{{ArmyID: "ours", OwnerID: "attacker", StartingTroops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 5}}}},
		Defenders: domain.BattleReportSide{UserIDs: []string{"defender"}, StartingMilitia: 20, SurvivingMilitia: 12, Armies: []domain.BattleReportArmy{{ArmyID: "theirs", OwnerID: "defender", StartingTroops: map[domain.TroopType]int64{domain.TroopTypeArcher: 8}}}, Settlement: &domain.BattleReportSettlement{CityID: cityID, StartingPopulation: 100, EndingPopulation: 95, CivilianCasualties: 5}},
		Rounds:    []domain.BattleReportRound{{AttackerPower: 50, DefenderPower: 120, DefenderLosses: []domain.BattleReportLoss{{MilitiaCityID: &cityID, Militia: 8}}, DefenderCivilianCasualties: 5}},
	}}

	report := MailboxMessageToProto(message).GetBattleReport()

	if !report.GetAttackers().GetStrengthVisible() || len(report.GetAttackers().GetArmies()[0].GetStartingTroops()) == 0 {
		t.Fatal("recipient's own strength was hidden")
	}
	if report.GetDefenders().GetStrengthVisible() || report.GetDefenders().GetStartingMilitia() != 0 || len(report.GetDefenders().GetArmies()[0].GetStartingTroops()) != 0 {
		t.Fatalf("opposing report strength was exposed: %+v", report.GetDefenders())
	}
	if report.GetRounds()[0].GetDefenderPower() != 0 || len(report.GetRounds()[0].GetDefenderLosses()) != 0 || report.GetDefenders().GetSettlement().GetCivilianCasualties() != 0 {
		t.Fatalf("opposing count-derived report detail was exposed: %+v", report)
	}
}

func TestHidePrivateArmyFieldsPreservesPhysicalState(t *testing.T) {
	destination := 4
	orderID := "order"
	army := ArmyToProto(domain.Army{
		ArmyID: "army", Owner: "owner", X: 2, Y: 3,
		Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 12},
		DestX:  &destination, DestY: &destination, OrderID: &orderID,
	})

	HidePrivateArmyFields(army)

	if army.GetArmyId().GetValue() != "army" || army.GetCoords().GetX() != 2 || army.GetCoords().GetY() != 3 {
		t.Fatalf("physical army state changed: %+v", army)
	}
	if army.GetCompositionVisibility() != entityv1.ArmyCompositionVisibility_ARMY_COMPOSITION_VISIBILITY_HIDDEN || len(army.GetTroops()) != 0 || army.GetOrderId() != nil {
		t.Fatalf("private army state was exposed: %+v", army)
	}
}

func TestHidePrivateCityFieldsHidesDemographicsAndStrength(t *testing.T) {
	city := CityToProto(domain.City{
		CityID: "city", Population: 250, PopulationCap: 250, PopulationBasis: 250,
		MilitiaPopulation: 25, MilitiaTarget: 25, TaxRatePercent: 20,
		TaxIncomeRate: 720, PopulationGrowthBeforeTaxRate: 24,
	})

	HidePrivateCityFields(city)

	if city.GetCityId().GetValue() != "city" || city.GetStart() == nil {
		t.Fatalf("settlement identity or location was hidden: %+v", city)
	}
	if city.GetDemographicsVisible() || city.GetPopulation() != 0 || city.GetPopulationCap() != 0 || city.GetMilitiaPopulation() != 0 || city.GetMilitiaTarget() != 0 || city.GetMilitiaPercent() != 0 || city.GetCorePopulation() != 0 || city.GetTaxablePopulation() != 0 {
		t.Fatalf("demographic or defensive state was exposed: %+v", city)
	}
	if city.GetPopulationGrowth() != nil || city.GetRecruitablePopulation() != 0 || city.GetCorePopulationFloor() != 0 || city.GetTaxRatePercent() != 0 || city.GetTaxIncome() != nil || city.GetPopulationGrowthBeforeTax() != nil {
		t.Fatalf("private economy state was exposed: %+v", city)
	}
}

func TestCityToProtoMarksOwnerProjectionDemographicsVisible(t *testing.T) {
	city := CityToProto(domain.City{CityID: "city", Population: 120, PopulationCap: 250})
	if !city.GetDemographicsVisible() {
		t.Fatal("full city projection did not disclose demographics")
	}
}

func TestTrainingOrderProjectionDistinguishesQueuedAndAssignedWork(t *testing.T) {
	queued := TrainingOrderToProto(domain.TrainingOrder{TrainingOrderID: "queued", ArmyID: "army", CityID: "city"})
	if queued.GetCityId().GetValue() != "city" || queued.GetBarracksId() != nil || queued.GetStartedAt() != nil {
		t.Fatalf("queued projection = %+v", queued)
	}
	barracksID := "barracks"
	assigned := TrainingOrderToProto(domain.TrainingOrder{TrainingOrderID: "active", ArmyID: "army", CityID: "city", BarracksID: &barracksID})
	if assigned.GetBarracksId().GetValue() != barracksID {
		t.Fatalf("assigned barracks = %+v", assigned.GetBarracksId())
	}
}

func TestMapTilesAroundPointToProtoClampsAndIncludesOccupancy(t *testing.T) {
	grid := domain.TerrainGrid{
		Width:  4,
		Height: 4,
		Tiles:  make([]domain.TerrainType, 16),
	}
	armies := []domain.Army{{ArmyID: "scout", X: 1, Y: 1}}

	tiles := MapTilesAroundPointToProto(grid, 0, 0, 1, nil, nil, armies)
	if len(tiles) != 4 {
		t.Fatalf("got %d tiles, want 4", len(tiles))
	}
	last := tiles[len(tiles)-1]
	if last.GetTileId().GetX() != 1 || last.GetTileId().GetY() != 1 {
		t.Fatalf("last tile = (%d,%d), want (1,1)", last.GetTileId().GetX(), last.GetTileId().GetY())
	}
	if len(last.GetArmyIds()) != 1 || last.GetArmyIds()[0].GetValue() != "scout" {
		t.Fatalf("army ids = %+v, want scout", last.GetArmyIds())
	}
}
