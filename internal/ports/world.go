package ports

import "cityio/internal/domain"

// WorldProvider exposes immutable terrain and allocates terrain-valid city sites.
type WorldProvider interface {
	Terrain() domain.TerrainGrid
	TerrainAt(x, y int) (domain.TerrainType, bool)
	ReserveCity(size int) (domain.Coordinates, error)
}
