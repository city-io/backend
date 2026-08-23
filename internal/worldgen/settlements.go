package worldgen

import (
	"math"
	mathrand "math/rand"

	"cityio/internal/constants"
	"cityio/internal/domain"
)

const (
	settlementGap        = 1
	capitalMinSeparation = 14
)

func placeCapitalSites(terrain domain.TerrainGrid, occupied []bool, config Config, rng *mathrand.Rand) []domain.Coordinates {
	sites := make([]domain.Coordinates, 0, config.CapitalSites)
	for len(sites) < config.CapitalSites {
		bestScore := -math.MaxFloat64
		var best domain.Coordinates
		found := false
		for y := 1; y+config.CapitalSize < config.Height; y++ {
			for x := 1; x+config.CapitalSize < config.Width; x++ {
				if !canPlace(terrain, occupied, x, y, config.CapitalSize, settlementGap) {
					continue
				}
				centerX := x + config.CapitalSize/2
				centerY := y + config.CapitalSize/2
				nearest := float64(max(config.Width, config.Height))
				for _, site := range sites {
					dx := centerX - (site.X + config.CapitalSize/2)
					dy := centerY - (site.Y + config.CapitalSize/2)
					distance := math.Hypot(float64(dx), float64(dy))
					nearest = min(nearest, distance)
				}
				if len(sites) > 0 && nearest < capitalMinSeparation {
					continue
				}
				score := footprintScore(terrain, x, y, config.CapitalSize)*3 + nearest + rng.Float64()*2
				if score > bestScore {
					bestScore = score
					best = domain.Coordinates{X: x, Y: y}
					found = true
				}
			}
		}
		if !found {
			break
		}
		sites = append(sites, best)
		markOccupied(occupied, config.Width, best.X, best.Y, config.CapitalSize)
	}
	return sites
}

func placeTowns(terrain domain.TerrainGrid, occupied []bool, config Config, rng *mathrand.Rand) []TownPlan {
	towns := make([]TownPlan, 0, config.TownTarget)
	usedNames := make(map[string]bool)
	for len(towns) < config.TownTarget {
		preferred := townSize(rng)
		sizes := []int{preferred, 2, 3, 4, 5}
		seen := make(map[int]bool)
		var placement domain.Coordinates
		placedSize := 0
		for _, size := range sizes {
			if seen[size] {
				continue
			}
			seen[size] = true
			candidate, ok := bestAvailablePlacement(terrain, occupied, size, rng)
			if ok {
				placement, placedSize = candidate, size
				break
			}
		}
		if placedSize == 0 {
			break
		}

		markOccupied(occupied, config.Width, placement.X, placement.Y, placedSize)
		towns = append(towns, buildTownPlan(rng, usedNames, placement.X, placement.Y, placedSize))
	}
	return towns
}

func bestAvailablePlacement(terrain domain.TerrainGrid, occupied []bool, size int, rng *mathrand.Rand) (domain.Coordinates, bool) {
	bestScore := -math.MaxFloat64
	var best domain.Coordinates
	found := false
	for y := 1; y+size < terrain.Height; y++ {
		for x := 1; x+size < terrain.Width; x++ {
			if !canPlace(terrain, occupied, x, y, size, settlementGap) {
				continue
			}
			score := footprintScore(terrain, x, y, size) + rng.Float64()*4
			if score > bestScore {
				bestScore = score
				best = domain.Coordinates{X: x, Y: y}
				found = true
			}
		}
	}
	return best, found
}

func canPlace(terrain domain.TerrainGrid, occupied []bool, x, y, size, gap int) bool {
	if x-gap < 0 || y-gap < 0 || x+size+gap > terrain.Width || y+size+gap > terrain.Height {
		return false
	}
	for py := y; py < y+size; py++ {
		for px := x; px < x+size; px++ {
			tile, _ := terrain.At(px, py)
			if !buildableTerrain(tile) {
				return false
			}
		}
	}
	for py := y - gap; py < y+size+gap; py++ {
		for px := x - gap; px < x+size+gap; px++ {
			if occupied[py*terrain.Width+px] {
				return false
			}
		}
	}
	return true
}

func markOccupied(occupied []bool, width, x, y, size int) {
	for py := y; py < y+size; py++ {
		for px := x; px < x+size; px++ {
			occupied[py*width+px] = true
		}
	}
}

func buildableTerrain(terrain domain.TerrainType) bool {
	return terrain != domain.TerrainTypeWater && terrain != domain.TerrainTypeMountains && terrain != domain.TerrainTypeMarsh
}

func footprintScore(terrain domain.TerrainGrid, x, y, size int) float64 {
	var score float64
	seen := make(map[domain.TerrainType]bool)
	for py := y; py < y+size; py++ {
		for px := x; px < x+size; px++ {
			tile, _ := terrain.At(px, py)
			seen[tile] = true
			switch tile {
			case domain.TerrainTypeGrassland:
				score += 4
			case domain.TerrainTypePlains:
				score += 3
			case domain.TerrainTypeForest:
				score += 2
			case domain.TerrainTypeHills:
				score++
			case domain.TerrainTypeDesert:
				score += 0.25
			}
		}
	}
	return score/float64(size*size) + float64(len(seen))*0.15
}

func townSize(rng *mathrand.Rand) int {
	value := rng.Intn(1000)
	switch {
	case value < 30:
		return 5
	case value < 100:
		return 4
	case value < 500:
		return 3
	default:
		return 2
	}
}

func buildTownPlan(rng *mathrand.Rand, usedNames map[string]bool, x, y, size int) TownPlan {
	config := constants.TownSizeConfig[size]
	centerX, centerY := x+size/2, y+size/2
	buildings := []BuildingPlan{{
		Type:  domain.BuildingTypeTownCenter,
		Level: config.CenterLevel,
		X:     centerX,
		Y:     centerY,
	}}
	for _, position := range randomPositions(rng, x, y, size, centerX, centerY, config.HouseCount) {
		buildings = append(buildings, BuildingPlan{
			Type:  domain.BuildingTypeHouse,
			Level: 1,
			X:     position.X,
			Y:     position.Y,
		})
	}
	return TownPlan{Name: townName(rng, usedNames), X: x, Y: y, Size: size, Buildings: buildings}
}

func randomPositions(rng *mathrand.Rand, originX, originY, size, centerX, centerY, count int) []domain.Coordinates {
	candidates := make([]domain.Coordinates, 0, size*size-1)
	for x := originX; x < originX+size; x++ {
		for y := originY; y < originY+size; y++ {
			if x != centerX || y != centerY {
				candidates = append(candidates, domain.Coordinates{X: x, Y: y})
			}
		}
	}
	rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	return candidates[:count]
}

var townPrefixes = []string{
	"Ash", "Birch", "Briar", "Cedar", "Copper", "Crow", "Dusk", "Elder",
	"Elm", "Ember", "Fern", "Flint", "Frost", "Gold", "Granite", "Hawk",
	"Hazel", "Heath", "Heron", "Holly", "Iron", "Ivy", "Lark", "Maple",
	"Marsh", "Moss", "Oak", "Pine", "Raven", "Reed", "Rose", "Rowan",
	"Shadow", "Silver", "Slate", "Stone", "Storm", "Thorn", "Willow", "Wolf",
}

var townSuffixes = []string{
	"bank", "barrow", "borough", "bridge", "brook", "bury", "cliff", "crest",
	"dale", "dell", "fall", "feld", "ford", "gate", "glen", "grove",
	"haven", "hill", "hollow", "keep", "mead", "moor", "mound", "point",
	"pool", "reach", "ridge", "shire", "stead", "vale", "wall", "watch",
	"well", "wick", "wood", "worth",
}

func townName(rng *mathrand.Rand, used map[string]bool) string {
	for {
		name := townPrefixes[rng.Intn(len(townPrefixes))] + townSuffixes[rng.Intn(len(townSuffixes))]
		if !used[name] {
			used[name] = true
			return name
		}
	}
}
