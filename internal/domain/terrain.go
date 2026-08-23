package domain

// TerrainType identifies the natural terrain of a map tile.
type TerrainType string

const (
	TerrainTypeGrassland TerrainType = "grassland"
	TerrainTypePlains    TerrainType = "plains"
	TerrainTypeForest    TerrainType = "forest"
	TerrainTypeHills     TerrainType = "hills"
	TerrainTypeMountains TerrainType = "mountains"
	TerrainTypeDesert    TerrainType = "desert"
	TerrainTypeMarsh     TerrainType = "marsh"
	TerrainTypeWater     TerrainType = "water"
)

// TerrainGrid stores terrain in row-major order.
type TerrainGrid struct {
	Width  int
	Height int
	Tiles  []TerrainType
}

// At returns the terrain at a coordinate and whether it is in bounds.
func (g TerrainGrid) At(x, y int) (TerrainType, bool) {
	if x < 0 || y < 0 || x >= g.Width || y >= g.Height {
		return "", false
	}
	return g.Tiles[y*g.Width+x], true
}
