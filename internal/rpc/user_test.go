package rpc

import (
	"testing"

	entityv1 "cityio/internal/gen/cityio/entity/v1"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/mapping"
)

func TestDiffProjectedStateSeparatesHiddenAndDeletedEntities(t *testing.T) {
	previous := &projectedState{snapshot: &servicev1.StateSnapshot{Entities: &entityv1.EntityBag{
		Cities: []*entityv1.City{{CityId: mapping.ToCityId("hidden-city")}},
		Armies: []*entityv1.Army{{ArmyId: mapping.ToArmyId("deleted-army")}},
	}}}
	current := &projectedState{
		snapshot:          &servicev1.StateSnapshot{Entities: &entityv1.EntityBag{}},
		existingCities:    map[string]struct{}{"hidden-city": {}},
		existingBuildings: map[string]struct{}{},
		existingArmies:    map[string]struct{}{},
	}

	delta := diffProjectedState(previous, current)
	if got := delta.GetHidden().GetCityIds(); len(got) != 1 || got[0].GetValue() != "hidden-city" {
		t.Fatalf("hidden cities = %v", got)
	}
	if got := delta.GetDeleted().GetArmyIds(); len(got) != 1 || got[0].GetValue() != "deleted-army" {
		t.Fatalf("deleted armies = %v", got)
	}
}

func TestDiffProjectedStateEmitsTileKnowledgeChanges(t *testing.T) {
	tileID := mapping.ToTileId(4, 5)
	previous := &projectedState{snapshot: &servicev1.StateSnapshot{
		Entities:       &entityv1.EntityBag{Tiles: []*entityv1.Tile{{TileId: tileID}}},
		TileVisibility: []*servicev1.TileVisibility{{TileId: tileID, State: servicev1.TileVisibilityState_TILE_VISIBILITY_STATE_EXPLORED}},
	}}
	current := &projectedState{snapshot: &servicev1.StateSnapshot{
		Entities:       &entityv1.EntityBag{Tiles: []*entityv1.Tile{{TileId: tileID, ArmyIds: []*entityv1.ArmyId{mapping.ToArmyId("army")}}}},
		TileVisibility: []*servicev1.TileVisibility{{TileId: tileID, State: servicev1.TileVisibilityState_TILE_VISIBILITY_STATE_VISIBLE}},
	}}

	delta := diffProjectedState(previous, current)
	if len(delta.GetUpserts().GetTiles()) != 1 {
		t.Fatalf("tile upserts = %d, want 1", len(delta.GetUpserts().GetTiles()))
	}
	if len(delta.GetTileVisibility()) != 1 || delta.GetTileVisibility()[0].GetState() != servicev1.TileVisibilityState_TILE_VISIBILITY_STATE_VISIBLE {
		t.Fatalf("visibility changes = %v", delta.GetTileVisibility())
	}
}

func TestDiffProjectedStateDeletesCompletedOrder(t *testing.T) {
	previous := &projectedState{snapshot: &servicev1.StateSnapshot{Entities: &entityv1.EntityBag{
		ArmyOrders: []*entityv1.ArmyOrder{{ArmyOrderId: mapping.ToArmyOrderId("completed")}},
	}}}
	current := &projectedState{snapshot: &servicev1.StateSnapshot{Entities: &entityv1.EntityBag{}}, existingOrders: map[string]struct{}{}}

	delta := diffProjectedState(previous, current)
	if got := delta.GetDeleted().GetArmyOrderIds(); len(got) != 1 || got[0].GetValue() != "completed" {
		t.Fatalf("deleted orders = %v", got)
	}
}

func TestDiffProjectedStateHidesStillActiveOrder(t *testing.T) {
	previous := &projectedState{snapshot: &servicev1.StateSnapshot{Entities: &entityv1.EntityBag{
		ArmyOrders: []*entityv1.ArmyOrder{{ArmyOrderId: mapping.ToArmyOrderId("restricted")}},
	}}}
	current := &projectedState{
		snapshot:       &servicev1.StateSnapshot{Entities: &entityv1.EntityBag{}},
		existingOrders: map[string]struct{}{"restricted": {}},
	}

	delta := diffProjectedState(previous, current)
	if got := delta.GetHidden().GetArmyOrderIds(); len(got) != 1 || got[0].GetValue() != "restricted" {
		t.Fatalf("hidden orders = %v", got)
	}
}
