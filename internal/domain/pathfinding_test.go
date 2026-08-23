package domain

import "testing"

func TestFindLandPathRoutesAroundWater(t *testing.T) {
	grid := terrainGrid(5, 5, TerrainTypeGrassland)
	for y := 0; y < grid.Height; y++ {
		grid.Tiles[y*grid.Width+2] = TerrainTypeWater
	}
	grid.Tiles[4*grid.Width+2] = TerrainTypeGrassland

	path, ok := FindLandPath(grid, Coordinates{X: 0, Y: 2}, Coordinates{X: 4, Y: 2})
	if !ok {
		t.Fatal("expected a route through the gap")
	}
	for _, coords := range path {
		terrain, _ := grid.At(coords.X, coords.Y)
		if terrain == TerrainTypeWater {
			t.Fatalf("path enters water at %+v", coords)
		}
	}
	if got := path[len(path)-1]; got != (Coordinates{X: 4, Y: 2}) {
		t.Fatalf("last step = %+v, want destination", got)
	}
}

func TestFindLandPathPrefersFasterTerrain(t *testing.T) {
	grid := terrainGrid(5, 3, TerrainTypeGrassland)
	for x := 1; x < 4; x++ {
		grid.Tiles[grid.Width+x] = TerrainTypeMountains
	}

	path, ok := FindLandPath(grid, Coordinates{X: 0, Y: 1}, Coordinates{X: 4, Y: 1})
	if !ok {
		t.Fatal("expected a route")
	}
	for _, coords := range path {
		terrain, _ := grid.At(coords.X, coords.Y)
		if terrain == TerrainTypeMountains {
			t.Fatalf("path used slower mountain tile at %+v", coords)
		}
	}
}

func TestFindLandPathRejectsWaterDestination(t *testing.T) {
	grid := terrainGrid(2, 1, TerrainTypeGrassland)
	grid.Tiles[1] = TerrainTypeWater

	if _, ok := FindLandPath(grid, Coordinates{}, Coordinates{X: 1}); ok {
		t.Fatal("expected water destination to be unreachable")
	}
}

func TestTerrainMovementCost(t *testing.T) {
	if got := TerrainMovementCost(TerrainTypeGrassland); got != 1 {
		t.Fatalf("grassland cost = %d, want 1", got)
	}
	if got := TerrainMovementCost(TerrainTypeMarsh); got != 2 {
		t.Fatalf("marsh cost = %d, want 2", got)
	}
	if got := TerrainMovementCost(TerrainTypeMountains); got != 3 {
		t.Fatalf("mountain cost = %d, want 3", got)
	}
	if got := TerrainMovementCost(TerrainTypeWater); got != 0 {
		t.Fatalf("water cost = %d, want impassable", got)
	}
}

func terrainGrid(width, height int, terrain TerrainType) TerrainGrid {
	tiles := make([]TerrainType, width*height)
	for i := range tiles {
		tiles[i] = terrain
	}
	return TerrainGrid{Width: width, Height: height, Tiles: tiles}
}
