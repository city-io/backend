package actors

import (
	"testing"

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
	world := movementTestWorld{grid: domain.TerrainGrid{
		Width:  3,
		Height: 1,
		Tiles:  []domain.TerrainType{domain.TerrainTypeGrassland, domain.TerrainTypeMarsh, domain.TerrainTypeMountains},
	}}
	state := &armyActor{baseActor: baseActor{World: world}}

	state.path = []domain.Coordinates{{X: 1}}
	if got := state.nextWaitTicks(); got != 1 {
		t.Fatalf("marsh wait = %d, want 1 extra tick", got)
	}

	state.path = []domain.Coordinates{{X: 2}}
	if got := state.nextWaitTicks(); got != 2 {
		t.Fatalf("mountain wait = %d, want 2 extra ticks", got)
	}
}
