package actors

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/contracts"
	"cityio/internal/domain"
	"cityio/internal/messages"
	"cityio/internal/stream"
)

type armyOperationTestStore struct {
	contracts.Store
	mu      sync.Mutex
	deleted []string
}

func (s *armyOperationTestStore) AddExploredTiles(context.Context, string, []domain.Coordinates) error {
	return nil
}

func (s *armyOperationTestStore) GetCitiesByOwner(context.Context, string) ([]domain.City, error) {
	return nil, nil
}

func (s *armyOperationTestStore) DeleteArmy(_ context.Context, armyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, armyID)
	return nil
}

func (*armyOperationTestStore) EnqueueArmy(domain.Army) {}

type armyOperationTestCluster struct {
	contracts.ClusterProvider
	request func(kind, identity string, message any) (any, error)
}

func (c *armyOperationTestCluster) Request(kind, identity string, message any) (any, error) {
	if c.request != nil {
		return c.request(kind, identity, message)
	}
	return nil, errors.New("unexpected request")
}

func (*armyOperationTestCluster) Tell(string, string, any) error { return nil }

func spawnArmyOperationTestActor(store contracts.Store, cluster contracts.ClusterProvider) (*actor.ActorSystem, *actor.PID) {
	system := actor.NewActorSystem()
	pid := system.Root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return &armyActor{baseActor: baseActor{Store: store, Cluster: cluster}}
	}))
	return system, pid
}

func requestArmyOperation(t *testing.T, system *actor.ActorSystem, pid *actor.PID, message any) any {
	t.Helper()
	result, err := system.Root.RequestFuture(pid, message, time.Second).Result()
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func expectNoArmyStreamUpdate(t *testing.T, updates <-chan stream.StateUpdate) {
	t.Helper()
	select {
	case update := <-updates:
		t.Fatalf("unexpected army stream update: %+v", update)
	case <-time.After(50 * time.Millisecond):
	}
}

type movementTestWorld struct {
	grid domain.TerrainGrid
}

func (w movementTestWorld) Terrain() domain.TerrainGrid {
	return w.grid
}

func (w movementTestWorld) TerrainAt(x, y int) (domain.TerrainType, bool) {
	return w.grid.At(x, y)
}

func (movementTestWorld) ReserveCity(int) (domain.Coordinates, error) {
	return domain.Coordinates{}, nil
}

func TestArmyWaitsForSlowTerrain(t *testing.T) {
	if constants.TroopMovementTickInterval != 250*time.Millisecond {
		t.Fatalf("movement tick = %s, want 250ms", constants.TroopMovementTickInterval)
	}
	world := movementTestWorld{grid: domain.TerrainGrid{
		Width:  3,
		Height: 1,
		Tiles:  []domain.TerrainType{domain.TerrainTypeGrassland, domain.TerrainTypeMarsh, domain.TerrainTypeMountains},
	}}
	state := &armyActor{
		baseActor: baseActor{World: world},
		Army: domain.Army{Troops: map[domain.TroopType]int64{
			domain.TroopTypeSoldier: 1,
		}},
	}

	state.path = []domain.Coordinates{{X: 1}}
	if got := state.currentStepDuration(); got != 3300*time.Millisecond {
		t.Fatalf("marsh duration = %s, want 3.3s", got)
	}

	state.path = []domain.Coordinates{{X: 2}}
	if got := state.currentStepDuration(); got != 4950*time.Millisecond {
		t.Fatalf("mountain duration = %s, want 4.95s", got)
	}
}

func TestArmyUsesSlowestTroopMovement(t *testing.T) {
	tests := []struct {
		name   string
		troops map[domain.TroopType]int64
		want   time.Duration
	}{
		{name: "cavalry", troops: map[domain.TroopType]int64{domain.TroopTypeCavalry: 10}, want: 825 * time.Millisecond},
		{name: "soldiers", troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 10}, want: 1650 * time.Millisecond},
		{name: "artillery", troops: map[domain.TroopType]int64{domain.TroopTypeArtillery: 10}, want: 2475 * time.Millisecond},
		{name: "mixed", troops: map[domain.TroopType]int64{domain.TroopTypeCavalry: 10, domain.TroopTypeArtillery: 1}, want: 2475 * time.Millisecond},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &armyActor{Army: domain.Army{Troops: test.troops}}
			if got := state.baseMovementDuration(); got != test.want {
				t.Fatalf("movement duration = %s, want %s", got, test.want)
			}
		})
	}
}

func TestArmyMergePreservesMovementProgressAtNewSlowestSpeed(t *testing.T) {
	world := movementTestWorld{grid: domain.TerrainGrid{
		Width: 1, Height: 1, Tiles: []domain.TerrainType{domain.TerrainTypeGrassland},
	}}
	state := &armyActor{
		baseActor: baseActor{World: world},
		Army: domain.Army{Troops: map[domain.TroopType]int64{
			domain.TroopTypeCavalry: 10,
		}},
		path:             []domain.Coordinates{{}},
		movementProgress: 250 * time.Millisecond,
	}
	state.Army.Troops[domain.TroopTypeArtillery] = 1
	if state.movementProgress != 250*time.Millisecond {
		t.Fatalf("movement progress = %s, want 250ms", state.movementProgress)
	}
	if remaining := state.currentStepDuration() - state.movementProgress; remaining != 2225*time.Millisecond {
		t.Fatalf("remaining movement = %s, want 2.225s", remaining)
	}
}

func TestArmyCarriesFractionalMovementProgressBetweenTiles(t *testing.T) {
	world := movementTestWorld{grid: domain.TerrainGrid{
		Width: 1, Height: 1, Tiles: []domain.TerrainType{domain.TerrainTypeGrassland},
	}}
	state := &armyActor{
		baseActor: baseActor{World: world},
		Army: domain.Army{Troops: map[domain.TroopType]int64{
			domain.TroopTypeSoldier: 1,
		}},
		path: []domain.Coordinates{{}},
	}
	moves := 0
	for range 33 {
		if state.advanceMovementClock() {
			moves++
		}
	}
	if moves != 5 || state.movementProgress != 0 {
		t.Fatalf("after 8.25s: moves=%d progress=%s, want 5 moves and no remainder", moves, state.movementProgress)
	}
}

func TestClearOrderClearsEveryMovementField(t *testing.T) {
	destination := 7
	orderID := "order"
	state := &armyActor{
		Army: domain.Army{DestX: &destination, DestY: &destination, OrderID: &orderID},
		path: []domain.Coordinates{{X: 1, Y: 1}}, movementProgress: 250 * time.Millisecond,
	}

	state.clearOrder()

	if state.Army.DestX != nil || state.Army.DestY != nil || state.Army.OrderID != nil {
		t.Fatalf("army movement fields were not cleared: %+v", state.Army)
	}
	if state.path != nil || state.movementProgress != 0 {
		t.Fatalf("actor movement state was not cleared: path=%v progress=%s", state.path, state.movementProgress)
	}
}

func TestSplitCompositionDetachesRequestedTroops(t *testing.T) {
	remaining, detached, err := splitComposition(
		map[domain.TroopType]int64{domain.TroopTypeSoldier: 10, domain.TroopTypeCavalry: 3},
		map[domain.TroopType]int64{domain.TroopTypeSoldier: 4, domain.TroopTypeCavalry: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if remaining[domain.TroopTypeSoldier] != 6 || remaining[domain.TroopTypeCavalry] != 2 {
		t.Fatalf("remaining = %v", remaining)
	}
	if detached[domain.TroopTypeSoldier] != 4 || detached[domain.TroopTypeCavalry] != 1 {
		t.Fatalf("detached = %v", detached)
	}
}

func TestSplitCompositionRejectsEntireArmy(t *testing.T) {
	_, _, err := splitComposition(
		map[domain.TroopType]int64{domain.TroopTypeSoldier: 5},
		map[domain.TroopType]int64{domain.TroopTypeSoldier: 5},
	)
	var invalid *messages.InvalidArmySplitError
	if !errors.As(err, &invalid) {
		t.Fatalf("error = %T, want InvalidArmySplitError", err)
	}
}

func TestSplitCompositionRejectsUnavailableTroops(t *testing.T) {
	_, _, err := splitComposition(
		map[domain.TroopType]int64{domain.TroopTypeArcher: 2},
		map[domain.TroopType]int64{domain.TroopTypeArcher: 3},
	)
	var insufficient *messages.InsufficientTroopsError
	if !errors.As(err, &insufficient) {
		t.Fatalf("error = %T, want InsufficientTroopsError", err)
	}
}

func TestSurrenderCleansUpBeforeReplyWithoutIntermediatePublish(t *testing.T) {
	store := &armyOperationTestStore{}
	cluster := &armyOperationTestCluster{}
	system, pid := spawnArmyOperationTestActor(store, cluster)
	owner := "surrender-owner"
	requestArmyOperation(t, system, pid, &messages.CreateArmyMessage{
		Army: domain.Army{
			ArmyID: "source-army",
			Owner:  owner,
			Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 4},
		},
		Restore: true,
	})
	updates, unsubscribe := stream.Subscribe(owner)
	defer unsubscribe()

	result := requestArmyOperation(t, system, pid, messages.SurrenderTroopsMessage{})
	response, ok := result.(*messages.SurrenderTroopsResponseMessage)
	if !ok {
		t.Fatalf("surrender response = %T, want SurrenderTroopsResponseMessage", result)
	}
	if response.Troops[domain.TroopTypeSoldier] != 4 {
		t.Fatalf("surrendered troops = %v, want 4 soldiers", response.Troops)
	}
	store.mu.Lock()
	deleted := append([]string(nil), store.deleted...)
	store.mu.Unlock()
	if len(deleted) != 1 || deleted[0] != "source-army" {
		t.Fatalf("deleted armies = %v, want source-army", deleted)
	}
	expectNoArmyStreamUpdate(t, updates)
}

func TestSplitPublishesOnlyFinalSourceComposition(t *testing.T) {
	store := &armyOperationTestStore{}
	var createMessage *messages.CreateArmyMessage
	cluster := &armyOperationTestCluster{request: func(kind, _ string, message any) (any, error) {
		if kind != "army" {
			return nil, errors.New("unexpected split request target")
		}
		var ok bool
		createMessage, ok = message.(*messages.CreateArmyMessage)
		if !ok {
			return nil, errors.New("unexpected split request message")
		}
		return messages.Ack{}, nil
	}}
	system, pid := spawnArmyOperationTestActor(store, cluster)
	owner := "split-owner"
	requestArmyOperation(t, system, pid, &messages.CreateArmyMessage{
		Army: domain.Army{
			ArmyID: "source-army",
			Owner:  owner,
			Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 10},
		},
		Restore: true,
	})
	updates, unsubscribe := stream.Subscribe(owner)
	defer unsubscribe()

	result := requestArmyOperation(t, system, pid, messages.SplitArmyMessage{
		Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 4},
	})
	response, ok := result.(*messages.SplitArmyResponseMessage)
	if !ok {
		t.Fatalf("split response = %T, want SplitArmyResponseMessage", result)
	}
	if createMessage == nil || !createMessage.SuppressPublish {
		t.Fatal("split army create did not suppress its intermediate publish")
	}
	if got := response.Source.Troops[domain.TroopTypeSoldier]; got != 6 {
		t.Fatalf("source soldier count = %d, want 6", got)
	}
	if got := response.Army.Troops[domain.TroopTypeSoldier]; got != 4 {
		t.Fatalf("split soldier count = %d, want 4", got)
	}
	select {
	case update := <-updates:
		if update.Army == nil || update.Army.ArmyID != "source-army" {
			t.Fatalf("split stream update = %+v, want source army", update)
		}
		if got := update.Army.Troops[domain.TroopTypeSoldier]; got != 6 {
			t.Fatalf("streamed source soldier count = %d, want 6", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for split stream update")
	}
	expectNoArmyStreamUpdate(t, updates)

	system.Root.Send(pid, messages.DeleteArmyMessage{})
}
