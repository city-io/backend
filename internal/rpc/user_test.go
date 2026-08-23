package rpc

import (
	"testing"

	"cityio/internal/domain"
	"cityio/internal/stream"
)

func TestStateUpdateToResponseSeparatesEntitiesAndDeletions(t *testing.T) {
	deletedBuildingID := "deleted-building"
	deletedArmyID := "deleted-army"
	building := domain.Building{BuildingID: "updated-building"}

	response := stateUpdateToResponse(stream.StateUpdate{
		Building:          &building,
		DeletedBuildingID: &deletedBuildingID,
		DeletedArmyID:     &deletedArmyID,
	})

	if len(response.GetEntities().GetBuildings()) != 1 {
		t.Fatalf("buildings = %d, want 1", len(response.GetEntities().GetBuildings()))
	}
	if got := response.GetEntities().GetBuildings()[0].GetBuildingId().GetValue(); got != building.BuildingID {
		t.Fatalf("building ID = %q, want %q", got, building.BuildingID)
	}
	if len(response.GetDeletedBuildingIds()) != 1 {
		t.Fatalf("deleted building IDs = %d, want 1", len(response.GetDeletedBuildingIds()))
	}
	if got := response.GetDeletedBuildingIds()[0].GetValue(); got != deletedBuildingID {
		t.Fatalf("deleted building ID = %q, want %q", got, deletedBuildingID)
	}
	if len(response.GetDeletedArmyIds()) != 1 {
		t.Fatalf("deleted army IDs = %d, want 1", len(response.GetDeletedArmyIds()))
	}
	if got := response.GetDeletedArmyIds()[0].GetValue(); got != deletedArmyID {
		t.Fatalf("deleted army ID = %q, want %q", got, deletedArmyID)
	}
}

func TestStateUpdateToResponseOmitsEmptyEntityBag(t *testing.T) {
	deletedArmyID := "deleted-army"

	response := stateUpdateToResponse(stream.StateUpdate{DeletedArmyID: &deletedArmyID})

	if response.Entities != nil {
		t.Fatal("entities should be nil for a deletion-only update")
	}
}
