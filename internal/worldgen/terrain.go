package worldgen

import (
	"math"
	"slices"

	"cityio/internal/domain"
)

const minTerrainRegionSize = 4

func generateTerrain(seed int64, width, height int) domain.TerrainGrid {
	elevation := make([]float64, width*height)
	moisture := make([]float64, width*height)
	minDimension := float64(min(width, height))
	edgeDepth := max(minDimension*0.16, 4)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			elevation[idx] = fractalNoise(uint64(deriveSeed(seed, 0x454c45564154494f)), float64(x), float64(y))
			moisture[idx] = fractalNoise(uint64(deriveSeed(seed, 0x4d4f495354555245)), float64(x)+19.5, float64(y)-11.25)

			edgeDistance := float64(min(x, y, width-1-x, height-1-y))
			edgeFactor := min(edgeDistance/edgeDepth, 1)
			elevation[idx] -= (1 - smooth(edgeFactor)) * 0.52
		}
	}

	waterLevel := percentile(elevation, 0.22)
	hillLevel := percentile(elevation, 0.76)
	mountainLevel := percentile(elevation, 0.91)
	marshLevel := percentile(elevation, 0.38)
	dryLevel := percentile(moisture, 0.18)
	grasslandLevel := percentile(moisture, 0.48)
	forestLevel := percentile(moisture, 0.72)
	tiles := make([]domain.TerrainType, width*height)

	for i, value := range elevation {
		switch {
		case value <= waterLevel:
			tiles[i] = domain.TerrainTypeWater
		case value >= mountainLevel:
			tiles[i] = domain.TerrainTypeMountains
		case value >= hillLevel:
			tiles[i] = domain.TerrainTypeHills
		case value <= marshLevel && moisture[i] >= forestLevel:
			tiles[i] = domain.TerrainTypeMarsh
		case moisture[i] <= dryLevel:
			tiles[i] = domain.TerrainTypeDesert
		case moisture[i] >= forestLevel:
			tiles[i] = domain.TerrainTypeForest
		case moisture[i] >= grasslandLevel:
			tiles[i] = domain.TerrainTypeGrassland
		default:
			tiles[i] = domain.TerrainTypePlains
		}
	}

	cleanupTerrainRegions(tiles, width, height, minTerrainRegionSize)
	for x := 0; x < width; x++ {
		tiles[x] = domain.TerrainTypeWater
		tiles[(height-1)*width+x] = domain.TerrainTypeWater
	}
	for y := 0; y < height; y++ {
		tiles[y*width] = domain.TerrainTypeWater
		tiles[y*width+width-1] = domain.TerrainTypeWater
	}

	return domain.TerrainGrid{Width: width, Height: height, Tiles: tiles}
}

func fractalNoise(seed uint64, x, y float64) float64 {
	return valueNoise(seed, x, y, 30)*0.52 +
		valueNoise(seed^0x9e3779b97f4a7c15, x, y, 15)*0.31 +
		valueNoise(seed^0xbf58476d1ce4e5b9, x, y, 7.5)*0.17
}

func valueNoise(seed uint64, x, y, scale float64) float64 {
	fx := x / scale
	fy := y / scale
	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	tx := smooth(fx - float64(x0))
	ty := smooth(fy - float64(y0))

	a := lerp(noiseValue(seed, x0, y0), noiseValue(seed, x0+1, y0), tx)
	b := lerp(noiseValue(seed, x0, y0+1), noiseValue(seed, x0+1, y0+1), tx)
	return lerp(a, b, ty)
}

func noiseValue(seed uint64, x, y int) float64 {
	value := seed ^ uint64(int64(x))*0x9e3779b97f4a7c15 ^ uint64(int64(y))*0xbf58476d1ce4e5b9
	value += 0x9e3779b97f4a7c15
	value = (value ^ (value >> 30)) * 0xbf58476d1ce4e5b9
	value = (value ^ (value >> 27)) * 0x94d049bb133111eb
	value ^= value >> 31
	return float64(value>>11) / float64(uint64(1)<<53)
}

func smooth(value float64) float64 {
	return value * value * (3 - 2*value)
}

func lerp(a, b, amount float64) float64 {
	return a + (b-a)*amount
}

func percentile(values []float64, fraction float64) float64 {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	return sorted[int(float64(len(sorted)-1)*fraction)]
}

func cleanupTerrainRegions(tiles []domain.TerrainType, width, height, minimum int) {
	for range 3 {
		visited := make([]bool, len(tiles))
		changed := false
		for start := range tiles {
			if visited[start] {
				continue
			}
			region := collectRegion(tiles, visited, width, height, start)
			if len(region) >= minimum {
				continue
			}
			replacement, ok := neighboringTerrain(tiles, width, height, region)
			if !ok {
				continue
			}
			for _, idx := range region {
				tiles[idx] = replacement
			}
			changed = true
		}
		if !changed {
			return
		}
	}
}

func collectRegion(tiles []domain.TerrainType, visited []bool, width, height, start int) []int {
	terrain := tiles[start]
	region := []int{start}
	visited[start] = true
	for i := 0; i < len(region); i++ {
		idx := region[i]
		x, y := idx%width, idx/width
		for _, offset := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			nx, ny := x+offset[0], y+offset[1]
			if nx < 0 || ny < 0 || nx >= width || ny >= height {
				continue
			}
			next := ny*width + nx
			if !visited[next] && tiles[next] == terrain {
				visited[next] = true
				region = append(region, next)
			}
		}
	}
	return region
}

func neighboringTerrain(tiles []domain.TerrainType, width, height int, region []int) (domain.TerrainType, bool) {
	regionSet := make(map[int]struct{}, len(region))
	for _, idx := range region {
		regionSet[idx] = struct{}{}
	}
	counts := make(map[domain.TerrainType]int)
	for _, idx := range region {
		x, y := idx%width, idx/width
		for ny := max(0, y-1); ny <= min(height-1, y+1); ny++ {
			for nx := max(0, x-1); nx <= min(width-1, x+1); nx++ {
				next := ny*width + nx
				if _, inside := regionSet[next]; !inside {
					counts[tiles[next]]++
				}
			}
		}
	}
	order := []domain.TerrainType{
		domain.TerrainTypeWater,
		domain.TerrainTypeGrassland,
		domain.TerrainTypePlains,
		domain.TerrainTypeForest,
		domain.TerrainTypeHills,
		domain.TerrainTypeMountains,
		domain.TerrainTypeDesert,
		domain.TerrainTypeMarsh,
	}
	var selected domain.TerrainType
	best := 0
	for _, terrain := range order {
		count := counts[terrain]
		if count > best {
			selected, best = terrain, count
		}
	}
	return selected, best > 0
}
