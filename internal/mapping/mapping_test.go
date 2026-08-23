package mapping

import (
	"testing"

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

func TestHidePrivateCityFieldsPreservesPublicMilitia(t *testing.T) {
	city := CityToProto(domain.City{
		CityID: "city", Population: 250, PopulationCap: 250,
		MilitiaPopulation: 25, MilitiaPercent: 10, TaxRatePercent: 20,
		TaxIncomeRate: 720,
	})

	HidePrivateCityFields(city)

	if city.GetMilitiaPopulation() != 25 || city.GetMilitiaPercent() != 10 {
		t.Fatalf("public militia state was hidden: %+v", city)
	}
	if city.GetRecruitablePopulation() != 0 || city.GetTaxRatePercent() != 0 || city.GetTaxIncome() != nil {
		t.Fatalf("private policy state was exposed: %+v", city)
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
