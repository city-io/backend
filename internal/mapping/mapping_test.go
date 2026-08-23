package mapping

import (
	"testing"

	"cityio/internal/domain"
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
