package actors

import (
	"context"
	"errors"
	"testing"
	"time"

	"cityio/internal/constants"
	"cityio/internal/contracts"
	"cityio/internal/domain"
	"cityio/internal/messages"
)

type cityTrainingStore struct {
	contracts.Store
	created  []domain.TrainingOrder
	assigned []domain.TrainingOrder
	deleted  []string
}

func (s *cityTrainingStore) CreateTrainingOrder(_ context.Context, order domain.TrainingOrder) error {
	s.created = append(s.created, order)
	return nil
}

func (s *cityTrainingStore) AssignTrainingOrder(_ context.Context, orderID, barracksID string, startedAt, completesAt time.Time) error {
	s.assigned = append(s.assigned, domain.TrainingOrder{
		TrainingOrderID: orderID,
		BarracksID:      &barracksID,
		StartedAt:       domain.NullTime{Time: &startedAt},
		CompletesAt:     domain.NullTime{Time: &completesAt},
	})
	return nil
}

func (s *cityTrainingStore) DeleteTrainingOrder(_ context.Context, orderID string) error {
	s.deleted = append(s.deleted, orderID)
	return nil
}

func (*cityTrainingStore) EnqueueCity(domain.City) {}

type cityTrainingCluster struct {
	contracts.ClusterProvider
	deducted  int64
	refunded  int64
	notices   int
	refundErr bool
}

func (c *cityTrainingCluster) Request(kind, _ string, message any) (any, error) {
	if kind != "user" {
		return nil, errors.New("unexpected request")
	}
	switch msg := message.(type) {
	case messages.CheckAndDeductGoldMessage:
		c.deducted += msg.Amount
		return messages.Ack{}, nil
	case messages.RefundUserGoldMessage:
		if c.refundErr {
			return nil, errors.New("temporary refund failure")
		}
		c.refunded += msg.Amount
		return messages.Ack{}, nil
	default:
		return nil, errors.New("unexpected user message")
	}
}

func (c *cityTrainingCluster) Tell(kind, _ string, message any) error {
	if kind == "building" {
		if _, ok := message.(messages.TrainingQueueAvailableMessage); ok {
			c.notices++
		}
	}
	return nil
}

func trainingCityState() (*cityActor, *cityTrainingStore, *cityTrainingCluster) {
	owner := "owner"
	store := &cityTrainingStore{}
	cluster := &cityTrainingCluster{}
	state := &cityActor{
		baseActor:      baseActor{Store: store, Cluster: cluster},
		City:           domain.City{CityID: "city", Owner: &owner, Population: 100, PopulationBasis: 100, PopulationCap: 100},
		trainingLoaded: true,
		barracksIDs:    map[string]struct{}{"barracks-1": {}, "barracks-2": {}},
		trainingOrders: nil,
		armyUpkeep:     make(map[string]int64),
		foodProduction: make(map[string]int64),
	}
	return state, store, cluster
}

func TestCityTrainingReservesCostBeforeQueueing(t *testing.T) {
	state, store, cluster := trainingCityState()
	order, err := state.trainTroops(messages.TrainTroopsMessage{Type: domain.TroopTypeSoldier, Count: 3})
	if err != nil {
		t.Fatal(err)
	}
	if state.City.Population != 97 {
		t.Fatalf("population = %.0f, want 97", state.City.Population)
	}
	if cluster.deducted != 150 {
		t.Fatalf("gold deducted = %d, want 150", cluster.deducted)
	}
	if len(store.created) != 1 || order.CityID != "city" || order.BarracksID != nil || order.StartedAt.Time != nil {
		t.Fatalf("created order = %+v, want unassigned city order", order)
	}
	if cluster.notices != 2 {
		t.Fatalf("barracks notices = %d, want 2", cluster.notices)
	}
}

func TestCityTrainingAssignsFIFOAcrossBarracks(t *testing.T) {
	state, store, _ := trainingCityState()
	state.trainingOrders = []domain.TrainingOrder{
		{TrainingOrderID: "first", CityID: "city", TroopType: domain.TroopTypeSoldier, Count: 2},
		{TrainingOrderID: "second", CityID: "city", TroopType: domain.TroopTypeArcher, Count: 2},
	}

	first, err := state.claimTrainingOrder(messages.ClaimTrainingOrderMessage{BarracksID: "barracks-1", Level: 1})
	if err != nil {
		t.Fatal(err)
	}
	second, err := state.claimTrainingOrder(messages.ClaimTrainingOrderMessage{BarracksID: "barracks-2", Level: 3})
	if err != nil {
		t.Fatal(err)
	}
	if first.TrainingOrderID != "first" || second.TrainingOrderID != "second" {
		t.Fatalf("assigned %s then %s, want FIFO", first.TrainingOrderID, second.TrainingOrderID)
	}
	if len(store.assigned) != 2 {
		t.Fatalf("assignments = %d, want 2", len(store.assigned))
	}
	firstDuration := first.CompletesAt.Time.Sub(*first.StartedAt.Time)
	secondBase := time.Duration(constants.GetTroopTrainingDuration(second.TroopType, second.Count)) * time.Second
	secondDuration := second.CompletesAt.Time.Sub(*second.StartedAt.Time)
	if firstDuration != 10*time.Second || secondDuration >= secondBase {
		t.Fatalf("durations = %s and %s (base %s), want level-specific speed", firstDuration, secondDuration, secondBase)
	}
}

func TestQueuedTrainingCancellationFullyRefunds(t *testing.T) {
	state, store, cluster := trainingCityState()
	state.City.Population = 97
	state.trainingOrders = []domain.TrainingOrder{{
		TrainingOrderID: "queued",
		CityID:          "city",
		PopulationCost:  3,
		GoldCost:        150,
	}}

	if err := state.cancelTrainingOrder("queued"); err != nil {
		t.Fatal(err)
	}
	if state.City.Population != 100 || cluster.refunded != 150 {
		t.Fatalf("refund population=%.0f gold=%d, want 100 and 150", state.City.Population, cluster.refunded)
	}
	if len(store.deleted) != 1 || len(state.trainingOrders) != 0 {
		t.Fatal("cancelled order was not removed")
	}
}

func TestActiveTrainingCannotBeCancelled(t *testing.T) {
	state, store, cluster := trainingCityState()
	now := time.Now()
	state.trainingOrders = []domain.TrainingOrder{{TrainingOrderID: "active", StartedAt: domain.NullTime{Time: &now}}}
	err := state.cancelTrainingOrder("active")
	var started *messages.TrainingAlreadyStartedError
	if !errors.As(err, &started) {
		t.Fatalf("error = %T, want TrainingAlreadyStartedError", err)
	}
	if len(store.deleted) != 0 || cluster.refunded != 0 {
		t.Fatal("active order was modified or refunded")
	}
}

func TestQueuedCancellationStaysQueuedUntilRefundSucceeds(t *testing.T) {
	state, store, cluster := trainingCityState()
	cluster.refundErr = true
	state.City.Population = 97
	state.trainingOrders = []domain.TrainingOrder{{
		TrainingOrderID: "queued",
		CityID:          "city",
		PopulationCost:  3,
		GoldCost:        150,
	}}

	if err := state.cancelTrainingOrder("queued"); err == nil {
		t.Fatal("cancel succeeded despite failed refund")
	}
	if state.City.Population != 97 || len(state.trainingOrders) != 1 {
		t.Fatal("failed refund partially cancelled the order")
	}
	if len(store.created) != 1 || store.created[0].TrainingOrderID != "queued" {
		t.Fatal("deleted order was not restored after refund failure")
	}
}
