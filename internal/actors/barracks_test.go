package actors

import (
	"errors"
	"testing"
	"time"

	"cityio/internal/contracts"
	"cityio/internal/domain"
	"cityio/internal/messages"
)

type trainingTestCluster struct {
	contracts.ClusterProvider
	spawnCalls int
	claimOrder *domain.TrainingOrder
}

func (c *trainingTestCluster) Request(kind, _ string, message any) (any, error) {
	switch kind {
	case "city":
		switch message.(type) {
		case messages.GetCityMessage:
			owner := "owner"
			return &messages.GetCityResponseMessage{City: domain.City{Owner: &owner}}, nil
		case messages.CompleteTrainingOrderMessage:
			return messages.Ack{}, nil
		case messages.ClaimTrainingOrderMessage:
			return &messages.ClaimTrainingOrderResponseMessage{Order: c.claimOrder}, nil
		}
	case "army":
		c.spawnCalls++
		if c.spawnCalls == 1 {
			return nil, errors.New("temporary spawn failure")
		}
		if _, ok := message.(*messages.CreateArmyMessage); !ok {
			return nil, errors.New("unexpected army message")
		}
		return messages.Ack{}, nil
	}
	return nil, errors.New("unexpected request")
}

func TestBarracksRetriesCompletedOrderUntilArmySpawns(t *testing.T) {
	completedAt := time.Now().Add(-time.Second)
	barracksID := "barracks"
	order := domain.TrainingOrder{
		TrainingOrderID: "order",
		ArmyID:          "army",
		CityID:          "city",
		BarracksID:      &barracksID,
		TroopType:       domain.TroopTypeSoldier,
		Count:           2,
		CompletesAt:     domain.NullTime{Time: &completedAt},
	}
	cluster := &trainingTestCluster{}
	state := &buildingActor{
		baseActor: baseActor{Cluster: cluster},
		Building:  domain.Building{BuildingID: barracksID, CityID: "city", X: 2, Y: 3, Level: 1, TargetLevel: 1},
	}
	barracks := &barracksImpl{active: &order}

	barracks.checkComplete(nil, state)
	if barracks.active == nil {
		t.Fatal("failed spawn removed the paid training order")
	}

	barracks.checkComplete(nil, state)
	if barracks.active != nil {
		t.Fatal("successful retry did not finish the training order")
	}
	if cluster.spawnCalls != 2 {
		t.Fatalf("spawn calls = %d, want 2", cluster.spawnCalls)
	}
}

func TestBarracksUpgradeRejectedOnlyWhileActivelyTraining(t *testing.T) {
	state := &buildingActor{
		Building: domain.Building{
			BuildingID:  "barracks",
			Type:        string(domain.BuildingTypeBarracks),
			Level:       1,
			TargetLevel: 1,
		},
		Impl: &barracksImpl{active: &domain.TrainingOrder{TrainingOrderID: "order"}},
	}

	err := state.upgrade(nil)
	var training *messages.TrainingInProgressError
	if !errors.As(err, &training) {
		t.Fatalf("upgrade error = %T, want TrainingInProgressError", err)
	}
}

func TestBarracksClaimsQueuedWorkBeforeUpgrade(t *testing.T) {
	order := &domain.TrainingOrder{TrainingOrderID: "queued"}
	cluster := &trainingTestCluster{claimOrder: order}
	state := &buildingActor{
		baseActor: baseActor{Cluster: cluster},
		Building: domain.Building{
			BuildingID:  "barracks",
			CityID:      "city",
			Level:       1,
			TargetLevel: 1,
		},
		Impl: &barracksImpl{},
	}

	err := state.upgrade(nil)
	var training *messages.TrainingInProgressError
	if !errors.As(err, &training) {
		t.Fatalf("upgrade error = %T, want TrainingInProgressError", err)
	}
	if state.Impl.(*barracksImpl).active != order {
		t.Fatal("idle barracks did not claim queued city work before upgrade")
	}
}
