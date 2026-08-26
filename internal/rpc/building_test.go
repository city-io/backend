package rpc

import (
	"testing"

	"cityio/internal/constants"
	"cityio/internal/domain"
)

func TestStandaloneStructurePlacementUsesSettlementFootprintDistance(t *testing.T) {
	city := domain.City{StartX: 20, StartY: 20, Size: 5}
	if !withinStructurePlacementRange([]domain.City{city}, 24+constants.StructurePlacementRadius, 24) {
		t.Fatal("tile at placement boundary should be allowed")
	}
	if withinStructurePlacementRange([]domain.City{city}, 24+constants.StructurePlacementRadius+1, 24) {
		t.Fatal("tile beyond placement boundary should be rejected")
	}
}

func TestStructureVisionPointsUseCompletedWatchtowerLevel(t *testing.T) {
	buildings := []domain.Building{
		{Owner: "owner", Type: string(domain.BuildingTypeWatchtower), Level: 2, TargetLevel: 3, X: 4, Y: 5},
		{Owner: "owner", Type: string(domain.BuildingTypeFort), Level: 1, TargetLevel: 1, X: 6, Y: 7},
		{Owner: "other", Type: string(domain.BuildingTypeWatchtower), Level: 10, TargetLevel: 10, X: 8, Y: 9},
	}

	points := structureVisionPoints("owner", buildings)
	if len(points) != 2 {
		t.Fatalf("got %d points, want 2", len(points))
	}
	if points[0].X != 4 || points[0].Y != 5 || points[0].Radius != constants.GetBuildingVisionRadius(domain.BuildingTypeWatchtower, 2) {
		t.Fatalf("watchtower point = %+v", points[0])
	}
	if points[1].Radius != 0 {
		t.Fatalf("fort vision radius = %d, want 0", points[1].Radius)
	}
}
