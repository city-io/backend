package actors

import (
	"context"
	"errors"
	"testing"
	"time"

	"cityio/internal/contracts"
	"cityio/internal/domain"
	"cityio/internal/messages"
)

type trainingTestStore struct {
	contracts.Store
	deleteCalls int
}

func (s *trainingTestStore) DeleteTrainingOrder(_ context.Context, _ string) error {
	s.deleteCalls++
	return nil
}

type trainingTestCluster struct {
	contracts.ClusterProvider
	spawnCalls int
}

func (c *trainingTestCluster) Request(kind, _ string, message any) (any, error) {
	switch kind {
	case "city":
		owner := "owner"
		return &messages.GetCityResponseMessage{City: domain.City{Owner: &owner}}, nil
	case "army":
		c.spawnCalls++
		if c.spawnCalls == 1 {
			return nil, errors.New("temporary spawn failure")
		}
		if _, ok := message.(*messages.CreateArmyMessage); !ok {
			return nil, errors.New("unexpected army message")
		}
		return messages.Ack{}, nil
	default:
		return nil, errors.New("unexpected request")
	}
}

func TestBarracksRetriesCompletedOrderUntilArmySpawns(t *testing.T) {
	completedAt := time.Now().Add(-time.Second)
	order := domain.TrainingOrder{
		TrainingOrderID: "order",
		ArmyID:          "army",
		BarracksID:      "barracks",
		TroopType:       domain.TroopTypeSoldier,
		Count:           2,
		CompletesAt:     domain.NullTime{Time: &completedAt},
	}
	store := &trainingTestStore{}
	cluster := &trainingTestCluster{}
	state := &buildingActor{
		baseActor: baseActor{Store: store, Cluster: cluster},
		Building:  domain.Building{BuildingID: "barracks", CityID: "city", X: 2, Y: 3},
	}
	barracks := &barracksImpl{queue: []domain.TrainingOrder{order}, loaded: true}

	barracks.checkComplete(nil, state)
	if len(barracks.queue) != 1 || store.deleteCalls != 0 {
		t.Fatal("failed spawn removed the paid training order")
	}

	barracks.checkComplete(nil, state)
	if len(barracks.queue) != 0 || store.deleteCalls != 1 {
		t.Fatal("successful retry did not finish the training order")
	}
	if cluster.spawnCalls != 2 {
		t.Fatalf("spawn calls = %d, want 2", cluster.spawnCalls)
	}
}

func TestBarracksUpgradeRejectedWhileTrainingPending(t *testing.T) {
	state := &buildingActor{
		Building: domain.Building{
			BuildingID:  "barracks",
			Type:        string(domain.BuildingTypeBarracks),
			Level:       1,
			TargetLevel: 1,
		},
		Impl: &barracksImpl{
			queue:  []domain.TrainingOrder{{TrainingOrderID: "order"}},
			loaded: true,
		},
	}

	err := state.upgrade(nil)
	var training *messages.TrainingInProgressError
	if !errors.As(err, &training) {
		t.Fatalf("upgrade error = %T, want TrainingInProgressError", err)
	}
}
