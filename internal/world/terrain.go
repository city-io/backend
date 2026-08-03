// Package world generates the game map: ground, landform and vegetation planes
// plus rivers and special resources. It is a pure package — no framework
// imports and no I/O — so setup, services and rpc can all depend on it the way
// they depend on domain.
package world

import (
	"math"
	"sort"
)

// Terrain is the ground cover of a tile.
type Terrain uint8

const (
	// The zero value is unused: it exists so these line up exactly with the
	// proto enums, whose STANDARD lint rules require an UNSPECIFIED zero. That
	// lets the planes be copied to the wire byte-for-byte with no remapping.
	TerrainUnspecified Terrain = iota
	DeepOcean
	Ocean
	Coast
	Lake
	Beach
	Grassland
	Plains
	Desert
	Tundra
	Snow
)

// Relief is the landform, drawn over the ground.
type Relief uint8

const (
	ReliefUnspecified Relief = iota
	Flat
	Hills
	Mountains
)

// Feature is vegetation or surface cover, drawn over both.
type Feature uint8

const (
	// NoFeature is the zero value and doubles as the proto UNSPECIFIED: a tile
	// with no feature specified simply has none.
	NoFeature Feature = iota
	Forest
	Jungle
	Marsh
	Oasis
	Ice
)

// Special is a bonus resource marker. Purely decorative for now.
type Special uint8

const (
	// NoSpecial is the zero value and doubles as the proto UNSPECIFIED.
	NoSpecial Special = iota
	Wheat
	Game
	Furs
	Fish
	Whales
	Coal
	Iron
	Gold
	Gems
)

// The world is three orthogonal planes rather than one flat list of biomes.
// Collapsed into one, forest-on-tundra and forest-on-plains would be separate
// values needing separate art; kept apart, a feature composites over any ground
// and the client's texture count stays small.
type World struct {
	Width  int
	Height int
	Seed   int64

	Terrain []uint8
	Relief  []uint8
	Feature []uint8
	Special []uint8
	// Rivers holds a 6-bit mask per tile: bit i means the river continues
	// toward neighbour i. Both tiles either side of a step carry the reciprocal
	// bit, so each draws its own half and rivers occupy no tile of their own.
	Rivers []uint8

	elevation   []float64
	moisture    []float64
	temperature []float64
	levels      levels
}

type levels struct {
	sea      float64
	hill     float64
	mountain float64
}

// Feature periods are in unit-hex widths (one hex is 2.0 across).
const (
	elevPeriod   = 20.0
	ridgePeriod  = 9.2
	ridgeWeight  = 0.26
	moistPeriod  = 25.0
	tempPeriod   = 32.0
	forestPeriod = 8.4
	rimPeriod    = 14.0
	warpAmp      = 0.3

	lacunarity = 2.0
	gain       = 0.5

	// landFraction is high for a Civ-style map on purpose. Towns are spread by
	// Poisson-disk sampling across the whole grid, so every point of ocean is a
	// candidate site rejected; a large continent with inland seas keeps the map
	// densely settled without drowning half the towns.
	landFraction = 0.70
	hillQuantile = 0.66
	mtnQuantile  = 0.92
	lakeMaxTiles = 40

	// Climate. Latitude is shaped by a cubic mix: a linear ramp buries both
	// poles under about nine rows of ice.
	poleMix    = 0.6
	lapse      = 0.3
	snowTemp   = 0.11
	tundraTemp = 0.26
	iceTemp    = 0.07

	// Moisture cutoffs are quantiles of land, so each reads directly as a share
	// of the continent: desert is the driest 22% of it.
	desertMoist = 0.22
	desertTemp  = 0.50
	plainsMoist = 0.52
	forestMoist = 0.42
	forestPatch = 0.52
	jungleMoist = 0.78
	jungleTemp  = 0.72
	marshMoist  = 0.88
)

func (w *World) Index(x, y int) int { return y*w.Width + x }

func (w *World) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < w.Width && y < w.Height
}

// IsWater reports whether a tile is ocean, coast or lake.
func (w *World) IsWater(x, y int) bool {
	return w.InBounds(x, y) && isWater(w.Terrain[w.Index(x, y)])
}

// Buildable reports whether a settlement can stand on a tile. Water, mountains
// and permanent ice are excluded.
func (w *World) Buildable(x, y int) bool {
	if !w.InBounds(x, y) {
		return false
	}
	i := w.Index(x, y)
	return isLand(w.Terrain[i]) && Relief(w.Relief[i]) != Mountains && Terrain(w.Terrain[i]) != Snow
}

// TerrainAt returns the three planes for a tile.
func (w *World) TerrainAt(x, y int) (Terrain, Relief, Feature) {
	i := w.Index(x, y)
	return Terrain(w.Terrain[i]), Relief(w.Relief[i]), Feature(w.Feature[i])
}

func isLand(t uint8) bool  { return Terrain(t) >= Beach }
func isWater(t uint8) bool { return Terrain(t) >= DeepOcean && Terrain(t) <= Lake }

// Generate builds a world deterministically from a seed.
func Generate(width, height int, seed int64) *World {
	n := width * height
	s := int(seed)
	w := &World{
		Width:       width,
		Height:      height,
		Seed:        seed,
		Terrain:     make([]uint8, n),
		Relief:      make([]uint8, n),
		Feature:     make([]uint8, n),
		Special:     make([]uint8, n),
		Rivers:      make([]uint8, n),
		elevation:   make([]float64, n),
		moisture:    make([]float64, n),
		temperature: make([]float64, n),
	}

	px := make([]float64, n)
	py := make([]float64, n)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			px[y*width+x], py[y*width+x] = hexPoint(x, y)
		}
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			wx, wy := warp(px[i]/elevPeriod, py[i]/elevPeriod, s+11, warpAmp)
			e := fbm(wx, wy, s+101, 6, 2.05, gain)
			// Ridges only bite ground that is already high, so ranges rise out
			// of the highlands instead of scarring the plains.
			r := ridged(px[i]/ridgePeriod, py[i]/ridgePeriod, s+601, 4)
			w.elevation[i] = (e + ridgeWeight*r*smoothstep((e-0.45)/0.35)) * w.rimFalloff(x, y, px[i], py[i], s)
		}
	}

	// Thresholds come from quantiles rather than fixed cutoffs, so retuning the
	// noise can't accidentally flood or drown the entire world.
	sea := quantile(w.elevation, 1-landFraction)
	landElev := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		if w.elevation[i] >= sea {
			landElev = append(landElev, w.elevation[i])
		}
	}
	w.levels = levels{sea: sea, hill: quantile(landElev, hillQuantile), mountain: quantile(landElev, mtnQuantile)}

	distToWater := w.bfsDistance(func(i int) bool { return w.elevation[i] < sea })

	// Moisture is rank-normalized so the biome cutoffs read as "the driest 22%
	// of land". Ranking over land only matters: water carries the full coastal
	// bonus, so including it shoves every land tile to the bottom.
	raw := make([]float64, n)
	for i := 0; i < n; i++ {
		m := fbm(px[i]/moistPeriod, py[i]/moistPeriod, s+907, 4, lacunarity, gain)
		if d := distToWater[i]; d >= 0 {
			m += math.Max(0, 0.22-0.03*float64(d))
		}
		raw[i] = m
	}
	w.moisture = rankNormalize(raw, func(i int) bool { return w.elevation[i] >= sea })

	forestNoise := make([]float64, n)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			u := math.Abs(2*float64(y)/math.Max(1, float64(height-1)) - 1)
			lat := 1 - (poleMix*u + (1-poleMix)*u*u*u)
			jitter := (fbm(px[i]/tempPeriod, py[i]/tempPeriod, s+1777, 3, lacunarity, gain) - 0.5) * 0.18
			w.temperature[i] = clamp01(lat*1.05 + jitter - w.aboveSea(w.elevation[i])*lapse)
			forestNoise[i] = fbm(px[i]/forestPeriod, py[i]/forestPeriod, s+2311, 3, lacunarity, gain)
		}
	}

	for i := 0; i < n; i++ {
		if w.elevation[i] < sea {
			w.Terrain[i] = uint8(Ocean)
			w.Relief[i] = uint8(Flat)
		} else {
			w.Terrain[i] = uint8(w.classifyGround(w.elevation[i], w.moisture[i], w.temperature[i]))
			w.Relief[i] = uint8(w.classifyRelief(w.elevation[i]))
		}
		w.Feature[i] = uint8(w.classifyFeature(Terrain(w.Terrain[i]), Relief(w.Relief[i]), w.elevation[i], w.moisture[i], w.temperature[i], forestNoise[i]))
	}

	w.despeckle()
	w.markLakes()
	w.tierWater()
	w.markBeaches()
	w.smoothGround()
	w.carveRivers(s)
	w.placeSpecials(s)
	return w
}

func (w *World) aboveSea(e float64) float64 {
	return clamp01((e - w.levels.sea) / math.Max(1e-6, 1-w.levels.sea))
}

// rimFalloff pulls the map's outer ring under sea level so the landmass is
// ringed by ocean. The width is noise-modulated; a constant one leaves a
// visibly rectangular coastline.
func (w *World) rimFalloff(x, y int, pxi, pyi float64, seed int) float64 {
	d := x
	for _, v := range []int{y, w.Width - 1 - x, w.Height - 1 - y} {
		if v < d {
			d = v
		}
	}
	width := 2 + 6*fbm(pxi/rimPeriod, pyi/rimPeriod, seed+55, 2, lacunarity, gain)
	return smoothstep(float64(d) / width)
}

func (w *World) classifyGround(e, m, t float64) Terrain {
	switch {
	case t < snowTemp:
		return Snow
	case t < tundraTemp:
		return Tundra
	case m < desertMoist && t > desertTemp:
		return Desert
	case m < plainsMoist:
		return Plains
	default:
		return Grassland
	}
}

func (w *World) classifyRelief(e float64) Relief {
	switch {
	case e >= w.levels.mountain:
		return Mountains
	case e >= w.levels.hill:
		return Hills
	default:
		return Flat
	}
}

func (w *World) classifyFeature(ground Terrain, relief Relief, e, m, t, forestN float64) Feature {
	if isWater(uint8(ground)) {
		if t < iceTemp {
			return Ice
		}
		return NoFeature
	}
	if relief == Mountains {
		return NoFeature
	}
	if relief == Flat && m > marshMoist && w.aboveSea(e) < 0.14 {
		return Marsh
	}
	if relief == Flat && t > jungleTemp && m > jungleMoist {
		return Jungle
	}
	if ground == Snow {
		return NoFeature
	}
	// A separate short-period field is what makes woodland form patches rather
	// than tracing the moisture contour exactly.
	if m > forestMoist && forestN > forestPatch && ground != Desert {
		return Forest
	}
	return NoFeature
}

// bfsDistance runs a multi-source breadth-first search over the hex grid,
// returning -1 where unreachable.
func (w *World) bfsDistance(isSource func(i int) bool) []int {
	n := w.Width * w.Height
	dist := make([]int, n)
	queue := make([]int, 0, n)
	for i := 0; i < n; i++ {
		dist[i] = -1
		if isSource(i) {
			dist[i] = 0
			queue = append(queue, i)
		}
	}
	for qi := 0; qi < len(queue); qi++ {
		i := queue[qi]
		w.forEachNeighbor(i%w.Width, i/w.Width, func(j, _, _, _ int) {
			if dist[j] != -1 {
				return
			}
			dist[j] = dist[i] + 1
			queue = append(queue, j)
		})
	}
	return dist
}

// despeckle removes one-tile islands and one-tile holes.
func (w *World) despeckle() {
	for pass := 0; pass < 2; pass++ {
		snap := make([]uint8, len(w.Terrain))
		copy(snap, w.Terrain)
		for y := 0; y < w.Height; y++ {
			for x := 0; x < w.Width; x++ {
				i := y*w.Width + x
				land, total := 0, 0
				counts := map[uint8]int{}
				w.forEachNeighbor(x, y, func(j, _, _, _ int) {
					total++
					if isLand(snap[j]) {
						land++
						counts[snap[j]]++
					}
				})
				switch {
				case isLand(snap[i]) && land == 0:
					w.Terrain[i] = uint8(Ocean)
					w.Relief[i] = uint8(Flat)
					w.Feature[i] = uint8(NoFeature)
				case !isLand(snap[i]) && total >= 5 && land == total:
					best, bestCount := uint8(Plains), -1
					for t, c := range counts {
						// Iteration order over a Go map is randomised, so break
						// ties on the terrain value to stay reproducible.
						if c > bestCount || (c == bestCount && t < best) {
							best, bestCount = t, c
						}
					}
					w.Terrain[i] = best
				}
			}
		}
	}
}

// markLakes turns small enclosed water bodies that never touch the map border
// into lakes.
func (w *World) markLakes() {
	seen := make([]bool, len(w.Terrain))
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			start := y*w.Width + x
			if seen[start] || isLand(w.Terrain[start]) {
				continue
			}
			comp := []int{start}
			seen[start] = true
			touchesBorder := false
			for qi := 0; qi < len(comp); qi++ {
				i := comp[qi]
				cx, cy := i%w.Width, i/w.Width
				if cx == 0 || cy == 0 || cx == w.Width-1 || cy == w.Height-1 {
					touchesBorder = true
				}
				w.forEachNeighbor(cx, cy, func(j, _, _, _ int) {
					if seen[j] || isLand(w.Terrain[j]) {
						return
					}
					seen[j] = true
					comp = append(comp, j)
				})
			}
			if !touchesBorder && len(comp) <= lakeMaxTiles {
				for _, i := range comp {
					w.Terrain[i] = uint8(Lake)
				}
			}
		}
	}
}

// tierWater grades open water outward from the shore: coast, ocean, deep ocean.
func (w *World) tierWater() {
	distToLand := w.bfsDistance(func(i int) bool { return isLand(w.Terrain[i]) })
	for i := range w.Terrain {
		if Terrain(w.Terrain[i]) == Lake || isLand(w.Terrain[i]) {
			continue
		}
		switch d := distToLand[i]; {
		case d == 1:
			w.Terrain[i] = uint8(Coast)
		case d >= 4 || d == -1:
			w.Terrain[i] = uint8(DeepOcean)
		default:
			w.Terrain[i] = uint8(Ocean)
		}
	}
}

func (w *World) markBeaches() {
	snap := make([]uint8, len(w.Terrain))
	copy(snap, w.Terrain)
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			i := y*w.Width + x
			if Relief(w.Relief[i]) != Flat || Feature(w.Feature[i]) != NoFeature {
				continue
			}
			t := Terrain(snap[i])
			if t != Grassland && t != Plains && t != Desert {
				continue
			}
			if w.temperature[i] < 0.32 || w.aboveSea(w.elevation[i]) > 0.05 {
				continue
			}
			coastal := false
			w.forEachNeighbor(x, y, func(j, _, _, _ int) {
				if Terrain(snap[j]) == Coast {
					coastal = true
				}
			})
			if coastal {
				w.Terrain[i] = uint8(Beach)
			}
		}
	}
}

// smoothGround runs one majority pass so biome edges read as regions rather
// than noise.
func (w *World) smoothGround() {
	snap := make([]uint8, len(w.Terrain))
	copy(snap, w.Terrain)
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			i := y*w.Width + x
			if isWater(snap[i]) || Terrain(snap[i]) == Beach {
				continue
			}
			counts := map[uint8]int{}
			total := 0
			w.forEachNeighbor(x, y, func(j, _, _, _ int) {
				if isWater(snap[j]) {
					return
				}
				total++
				counts[snap[j]]++
			})
			if total < 5 {
				continue
			}
			for t, c := range counts {
				if t != snap[i] && c >= 5 {
					w.Terrain[i] = t
				}
			}
		}
	}
}

// carveRivers walks rivers down the distance-to-water gradient, tie-broken by
// elevation. Pure steepest descent on noise strands most rivers in local
// minima; steering by distance-to-water guarantees they reach it.
func (w *World) carveRivers(seed int) {
	distToWater := w.bfsDistance(func(i int) bool { return isWater(w.Terrain[i]) })

	candidates := make([]int, 0, len(w.Terrain))
	for i := range w.Terrain {
		if !isLand(w.Terrain[i]) || Terrain(w.Terrain[i]) == Snow {
			continue
		}
		if w.moisture[i] < 0.45 || distToWater[i] < 3 {
			continue
		}
		candidates = append(candidates, i)
	}
	// Highest ground first, so sources sit near watersheds.
	sort.Slice(candidates, func(a, b int) bool { return w.elevation[candidates[a]] > w.elevation[candidates[b]] })

	maxSources := (w.Width * w.Height) / 220
	if maxSources < 6 {
		maxSources = 6
	}
	const minSpacing = 6
	sources := make([]int, 0, maxSources)
	for _, i := range candidates {
		if len(sources) >= maxSources {
			break
		}
		x, y := i%w.Width, i/w.Width
		tooClose := false
		for _, s := range sources {
			sx, sy := s%w.Width, s/w.Width
			if (sx-x)*(sx-x)+(sy-y)*(sy-y) < minSpacing*minSpacing {
				tooClose = true
				break
			}
		}
		if !tooClose {
			sources = append(sources, i)
		}
	}

	for _, start := range sources {
		i := start
		visited := map[int]bool{i: true}
		for step := 0; step < 200; step++ {
			x, y := i%w.Width, i/w.Width
			bestJ, bestDir, bestD, bestE := -1, -1, 1<<30, math.Inf(1)
			w.forEachNeighbor(x, y, func(j, dir, _, _ int) {
				d := distToWater[j]
				if d < 0 || visited[j] {
					return
				}
				if d < bestD || (d == bestD && w.elevation[j] < bestE) {
					bestD, bestE, bestJ, bestDir = d, w.elevation[j], j, dir
				}
			})
			if bestJ < 0 {
				break
			}
			w.Rivers[i] |= 1 << uint(bestDir)
			w.Rivers[bestJ] |= 1 << uint(opposite(bestDir))
			visited[bestJ] = true
			if isWater(w.Terrain[bestJ]) {
				break
			}
			i = bestJ
		}
	}
}

type specialRule struct {
	kind   Special
	chance float64
	ok     func(g Terrain, r Relief, f Feature) bool
}

var specialRules = []specialRule{
	{Gold, 1.0 / 30, func(g Terrain, r Relief, f Feature) bool { return r == Mountains }},
	{Gems, 1.0 / 45, func(g Terrain, r Relief, f Feature) bool { return f == Jungle }},
	{Coal, 1.0 / 38, func(g Terrain, r Relief, f Feature) bool { return r == Hills }},
	{Iron, 1.0 / 42, func(g Terrain, r Relief, f Feature) bool { return r == Hills || r == Mountains }},
	{Furs, 1.0 / 40, func(g Terrain, r Relief, f Feature) bool { return f == Forest && (g == Tundra || g == Snow) }},
	{Game, 1.0 / 45, func(g Terrain, r Relief, f Feature) bool { return f == Forest }},
	{Wheat, 1.0 / 55, func(g Terrain, r Relief, f Feature) bool {
		return (g == Grassland || g == Plains) && r == Flat && f == NoFeature
	}},
	{Fish, 1.0 / 50, func(g Terrain, r Relief, f Feature) bool { return g == Coast }},
	{Whales, 1.0 / 80, func(g Terrain, r Relief, f Feature) bool { return g == Ocean }},
}

// placeSpecials makes one deterministic row-major pass. Rejecting a tile whose
// neighbour already carries a special keeps resources spread out; the fixed
// scan order is what makes that rule reproducible.
func (w *World) placeSpecials(seed int) {
	for y := 0; y < w.Height; y++ {
		for x := 0; x < w.Width; x++ {
			i := y*w.Width + x
			crowded := false
			w.forEachNeighbor(x, y, func(j, _, _, _ int) {
				if w.Special[j] != 0 {
					crowded = true
				}
			})
			if crowded {
				continue
			}
			roll := hash01(x, y, seed+4242)
			acc := 0.0
			g, r, f := Terrain(w.Terrain[i]), Relief(w.Relief[i]), Feature(w.Feature[i])
			for _, rule := range specialRules {
				if !rule.ok(g, r, f) {
					continue
				}
				acc += rule.chance
				if roll < acc {
					w.Special[i] = uint8(rule.kind)
					break
				}
			}
		}
	}
}
