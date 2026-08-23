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
	if got := state.nextWaitTicks(); got != 7 {
		t.Fatalf("marsh wait = %d, want 7 extra ticks", got)
	}

	state.path = []domain.Coordinates{{X: 2}}
	if got := state.nextWaitTicks(); got != 11 {
		t.Fatalf("mountain wait = %d, want 11 extra ticks", got)
	}
}

func TestArmyUsesSlowestTroopMovement(t *testing.T) {
	tests := []struct {
		name   string
		troops map[domain.TroopType]int64
		want   int
	}{
		{name: "cavalry", troops: map[domain.TroopType]int64{domain.TroopTypeCavalry: 10}, want: 2},
		{name: "soldiers", troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 10}, want: 4},
		{name: "artillery", troops: map[domain.TroopType]int64{domain.TroopTypeArtillery: 10}, want: 6},
		{name: "mixed", troops: map[domain.TroopType]int64{domain.TroopTypeCavalry: 10, domain.TroopTypeArtillery: 1}, want: 6},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := &armyActor{Army: domain.Army{Troops: test.troops}}
			if got := state.baseMovementTicks(); got != test.want {
				t.Fatalf("movement ticks = %d, want %d", got, test.want)
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
		path:      []domain.Coordinates{{}},
		waitTicks: 0,
	}
	elapsed := state.elapsedStepTicks()
	state.Army.Troops[domain.TroopTypeArtillery] = 1
	state.waitTicks = max(state.currentStepTicks()-elapsed-1, 0)

	if state.waitTicks != 4 {
		t.Fatalf("remaining wait = %d, want 4 after one elapsed tick", state.waitTicks)
	}
}
