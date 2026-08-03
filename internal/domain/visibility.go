package domain

// Watcher is an axis-aligned box that grants vision out to Radius tiles beyond
// its own edges. A city watches its footprint; an army watches the single tile
// it stands on. The radius rides on the watcher because it varies per source —
// a cavalry-led army scouts far, a lone city sees modestly — so a single global
// radius can no longer describe what a player can see.
//
// Vision used to be computed from owned cities at one fixed radius. That left
// armies unable to see at all and meant a player's view never changed after
// founding.
type Watcher struct {
	X, Y   int
	W, H   int
	Radius int
}

// NewCityWatcher builds a watcher over a city's footprint.
func NewCityWatcher(c City, radius int) Watcher {
	return Watcher{X: c.StartX, Y: c.StartY, W: c.Size, H: c.Size, Radius: radius}
}

// NewArmyWatcher builds a watcher over the single tile an army occupies.
func NewArmyWatcher(a Army, radius int) Watcher {
	return Watcher{X: a.X, Y: a.Y, W: 1, H: 1, Radius: radius}
}

// DistanceTo returns the Chebyshev (king-move) distance from a point to this
// box. Zero when the point lies inside it.
func (w Watcher) DistanceTo(px, py int) int {
	dx := max(0, w.X-px, px-(w.X+w.W-1))
	dy := max(0, w.Y-py, py-(w.Y+w.H-1))
	return max(dx, dy)
}

// PointVisible reports whether (px, py) falls within any watcher's radius.
func PointVisible(ws []Watcher, px, py int) bool {
	for _, w := range ws {
		if w.DistanceTo(px, py) <= w.Radius {
			return true
		}
	}
	return false
}

// BoxVisible reports whether any tile of target falls within any watcher's
// radius. Uses AABB overlap: expand each watcher by its radius and intersect.
func BoxVisible(ws []Watcher, target Watcher) bool {
	tx2, ty2 := target.X+target.W-1, target.Y+target.H-1
	for _, w := range ws {
		if w.X-w.Radius <= tx2 && w.X+w.W-1+w.Radius >= target.X &&
			w.Y-w.Radius <= ty2 && w.Y+w.H-1+w.Radius >= target.Y {
			return true
		}
	}
	return false
}

// CityVisible reports whether any tile of target is visible.
func CityVisible(ws []Watcher, target City) bool {
	return BoxVisible(ws, Watcher{X: target.StartX, Y: target.StartY, W: target.Size, H: target.Size})
}

// FilterCities returns the subset of all visible to the given watchers.
func FilterCities(ws []Watcher, all []City) []City {
	out := make([]City, 0, len(all))
	for _, c := range all {
		if CityVisible(ws, c) {
			out = append(out, c)
		}
	}
	return out
}

// FilterBuildings returns the subset of buildings visible to the given watchers.
func FilterBuildings(ws []Watcher, all []Building) []Building {
	out := make([]Building, 0, len(all))
	for _, b := range all {
		if PointVisible(ws, b.X, b.Y) {
			out = append(out, b)
		}
	}
	return out
}

// FilterArmies returns the subset of armies visible to the given watchers.
func FilterArmies(ws []Watcher, all []Army) []Army {
	out := make([]Army, 0, len(all))
	for _, a := range all {
		if PointVisible(ws, a.X, a.Y) {
			out = append(out, a)
		}
	}
	return out
}

// EachVisibleTile calls fn for every in-bounds tile within any watcher's
// radius, possibly more than once where watchers overlap. Walks each watcher's
// own neighbourhood rather than scanning the map, so the cost is proportional
// to what a player can see rather than to the size of the world.
func EachVisibleTile(ws []Watcher, width, height int, fn func(x, y int)) {
	for _, w := range ws {
		for y := w.Y - w.Radius; y < w.Y+w.H+w.Radius; y++ {
			for x := w.X - w.Radius; x < w.X+w.W+w.Radius; x++ {
				if x < 0 || y < 0 || x >= width || y >= height {
					continue
				}
				fn(x, y)
			}
		}
	}
}

// ChebyshevToCity returns the Chebyshev (king-move) distance from point (px, py)
// to the nearest tile of city c. Zero when the point lies inside the city box.
func ChebyshevToCity(c City, px, py int) int {
	return Watcher{X: c.StartX, Y: c.StartY, W: c.Size, H: c.Size}.DistanceTo(px, py)
}
