package rpc

import (
	"errors"
	"testing"

	"cityio/internal/constants"
	"cityio/internal/contracts"
	"cityio/internal/domain"
	"cityio/internal/messages"
)

type structurePlacementTestCluster struct {
	contracts.ClusterProvider
	armies map[string]domain.Army
}

func (c *structurePlacementTestCluster) Request(kind, identity string, message any) (any, error) {
	if kind != "army" {
		return nil, errors.New("unexpected actor kind")
	}
	if _, ok := message.(messages.GetArmyMessage); !ok {
		return nil, errors.New("unexpected army message")
	}
	army, ok := c.armies[identity]
	if !ok {
		return nil, errors.New("army not found")
	}
	return &messages.GetArmyResponseMessage{Army: army}, nil
}

func TestStandaloneStructurePlacementUsesSettlementFootprintDistance(t *testing.T) {
	city := domain.City{StartX: 20, StartY: 20, Size: 5}
	if !withinStructurePlacementRange([]domain.City{city}, 24+constants.StructurePlacementRadius, 24) {
		t.Fatal("tile at placement boundary should be allowed")
	}
	if withinStructurePlacementRange([]domain.City{city}, 24+constants.StructurePlacementRadius+1, 24) {
		t.Fatal("tile beyond placement boundary should be rejected")
	}
}

func TestStandaloneStructurePlacementRequiresOwnedArmyOnTile(t *testing.T) {
	cluster := &structurePlacementTestCluster{armies: map[string]domain.Army{
		"foreign": {ArmyID: "foreign", Owner: "other", X: 7, Y: 8},
		"moved":   {ArmyID: "moved", Owner: "owner", X: 8, Y: 8},
		"owned":   {ArmyID: "owned", Owner: "owner", X: 7, Y: 8},
	}}
	handler := &buildingHandler{srv: &Server{cluster: cluster}}

	got, err := handler.ownedArmyOccupiesTile("owner", 7, 8, nil)
	if err != nil || got {
		t.Fatalf("empty tile qualified: got %t, error %v", got, err)
	}
	got, err = handler.ownedArmyOccupiesTile("owner", 7, 8, []string{"foreign", "moved"})
	if err != nil || got {
		t.Fatalf("foreign or moved armies qualified: got %t, error %v", got, err)
	}
	got, err = handler.ownedArmyOccupiesTile("owner", 7, 8, []string{"foreign", "owned"})
	if err != nil || !got {
		t.Fatalf("owned army on tile did not qualify: got %t, error %v", got, err)
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
