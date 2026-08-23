package actors

import (
	"testing"
	"time"

	"cityio/internal/constants"
	"cityio/internal/domain"
)

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
	if got := state.currentStepDuration(); got != 2200*time.Millisecond {
		t.Fatalf("marsh duration = %s, want 2.2s", got)
	}

	state.path = []domain.Coordinates{{X: 2}}
	if got := state.currentStepDuration(); got != 3300*time.Millisecond {
		t.Fatalf("mountain duration = %s, want 3.3s", got)
	}
}

func TestArmyUsesSlowestTroopMovement(t *testing.T) {
	tests := []struct {
		name   string
		troops map[domain.TroopType]int64
		want   time.Duration
	}{
		{name: "cavalry", troops: map[domain.TroopType]int64{domain.TroopTypeCavalry: 10}, want: 550 * time.Millisecond},
		{name: "soldiers", troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 10}, want: 1100 * time.Millisecond},
		{name: "artillery", troops: map[domain.TroopType]int64{domain.TroopTypeArtillery: 10}, want: 1650 * time.Millisecond},
		{name: "mixed", troops: map[domain.TroopType]int64{domain.TroopTypeCavalry: 10, domain.TroopTypeArtillery: 1}, want: 1650 * time.Millisecond},
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
	if remaining := state.currentStepDuration() - state.movementProgress; remaining != 1400*time.Millisecond {
		t.Fatalf("remaining movement = %s, want 1.4s", remaining)
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
	for range 22 {
		if state.advanceMovementClock() {
			moves++
		}
	}
	if moves != 5 || state.movementProgress != 0 {
		t.Fatalf("after 5.5s: moves=%d progress=%s, want 5 moves and no remainder", moves, state.movementProgress)
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
