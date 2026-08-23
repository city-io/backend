// Package worldgen creates deterministic terrain and settlement plans from a seed.
package worldgen

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	mathrand "math/rand"
	"sync"

	"cityio/internal/domain"
)

// Config controls the dimensions and settlement density of a generated world.
type Config struct {
	Seed         int64
	Width        int
	Height       int
	CapitalSize  int
	CapitalSites int
	TownTarget   int
}

// BuildingPlan describes a prebuilt structure in a generated neutral town.
type BuildingPlan struct {
	Type  domain.BuildingType
	Level int
	X     int
	Y     int
}

// TownPlan describes a generated neutral town and its initial buildings.
type TownPlan struct {
	Name      string
	X         int
	Y         int
	Size      int
	Buildings []BuildingPlan
}

// World is the terrain and settlement allocation state for one generated map.
type World struct {
	seed        int64
	config      Config
	terrain     domain.TerrainGrid
	towns       []TownPlan
	capitalSite []domain.Coordinates

	mu          sync.Mutex
	nextCapital int
	occupied    []bool
	rng         *mathrand.Rand
}

// RandomSeed returns a seed suitable for generating a new world.
func RandomSeed() (int64, error) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(raw[:]) & (1<<63 - 1)), nil
}

// Generate creates a complete terrain grid, capital reservations, and neutral towns.
func Generate(config Config) (*World, error) {
	if config.Width < 16 || config.Height < 16 {
		return nil, errors.New("world dimensions must be at least 16 tiles")
	}
	if config.CapitalSize < 1 || config.CapitalSites < 1 || config.TownTarget < 0 {
		return nil, errors.New("invalid settlement generation config")
	}

	terrain := generateTerrain(config.Seed, config.Width, config.Height)
	occupied := make([]bool, config.Width*config.Height)
	placementRNG := mathrand.New(mathrand.NewSource(deriveSeed(config.Seed, 0x504c414345)))

	capitalSites := placeCapitalSites(terrain, occupied, config, placementRNG)
	if len(capitalSites) < config.CapitalSites {
		return nil, fmt.Errorf("placed %d of %d capital sites", len(capitalSites), config.CapitalSites)
	}

	towns := placeTowns(terrain, occupied, config, placementRNG)

	return &World{
		seed:        config.Seed,
		config:      config,
		terrain:     terrain,
		towns:       towns,
		capitalSite: capitalSites,
		occupied:    occupied,
		rng:         mathrand.New(mathrand.NewSource(deriveSeed(config.Seed, 0x52554e54494d45))),
	}, nil
}

// Seed returns the seed used to generate the world.
func (w *World) Seed() int64 {
	return w.seed
}

// Terrain returns a copy of the generated terrain grid.
func (w *World) Terrain() domain.TerrainGrid {
	tiles := append([]domain.TerrainType(nil), w.terrain.Tiles...)
	return domain.TerrainGrid{Width: w.terrain.Width, Height: w.terrain.Height, Tiles: tiles}
}

// TerrainAt returns the terrain at a coordinate and whether it is in bounds.
func (w *World) TerrainAt(x, y int) (domain.TerrainType, bool) {
	return w.terrain.At(x, y)
}

// Towns returns a copy of the generated neutral town plans.
func (w *World) Towns() []TownPlan {
	towns := make([]TownPlan, len(w.towns))
	for i, town := range w.towns {
		towns[i] = town
		towns[i].Buildings = append([]BuildingPlan(nil), town.Buildings...)
	}
	return towns
}

// ReserveCity allocates a terrain-valid, non-overlapping city footprint.
func (w *World) ReserveCity(size int) (domain.Coordinates, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if size < 1 {
		return domain.Coordinates{}, errors.New("city size must be positive")
	}

	if size == w.config.CapitalSize && w.nextCapital < len(w.capitalSite) {
		coords := w.capitalSite[w.nextCapital]
		w.nextCapital++
		return coords, nil
	}

	placement, ok := bestAvailablePlacement(w.terrain, w.occupied, size, w.rng)
	if !ok {
		return domain.Coordinates{}, errors.New("no terrain-valid city site available")
	}
	markOccupied(w.occupied, w.config.Width, placement.X, placement.Y, size)
	return placement, nil
}

func deriveSeed(seed int64, stream uint64) int64 {
	value := uint64(seed) ^ stream
	value += 0x9e3779b97f4a7c15
	value = (value ^ (value >> 30)) * 0xbf58476d1ce4e5b9
	value = (value ^ (value >> 27)) * 0x94d049bb133111eb
	return int64((value ^ (value >> 31)) & (1<<63 - 1))
}
