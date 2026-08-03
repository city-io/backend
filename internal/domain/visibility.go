package domain

// Watcher is an axis-aligned box that grants vision to whoever owns it. A city
// watches its whole footprint; an army watches the single tile it stands on.
//
// Vision used to be computed from owned cities alone. That left armies able to
// march anywhere without revealing a thing, and — more importantly — meant a
// player's visible area never changed after founding, so there was nothing to
// explore and no way to learn the shape of the map.
type Watcher struct {
	X, Y int
	W, H int
}

func CityWatcher(c City) Watcher {
	return Watcher{X: c.StartX, Y: c.StartY, W: c.Size, H: c.Size}
}

func ArmyWatcher(a Army) Watcher {
	return Watcher{X: a.X, Y: a.Y, W: 1, H: 1}
}

// WatchersFor builds a player's vision set from everything they own.
func WatchersFor(cities []City, armies []Army) []Watcher {
	ws := make([]Watcher, 0, len(cities)+len(armies))
	for i := range cities {
		ws = append(ws, CityWatcher(cities[i]))
	}
	for i := range armies {
		ws = append(ws, ArmyWatcher(armies[i]))
	}
	return ws
}

// DistanceTo returns the Chebyshev (king-move) distance from a point to this
// box. Zero when the point lies inside it.
func (w Watcher) DistanceTo(px, py int) int {
	dx := max(0, w.X-px, px-(w.X+w.W-1))
	dy := max(0, w.Y-py, py-(w.Y+w.H-1))
	return max(dx, dy)
}

// PointVisible reports whether (px, py) is within radius of any watcher.
func PointVisible(ws []Watcher, px, py, radius int) bool {
	for _, w := range ws {
		if w.DistanceTo(px, py) <= radius {
			return true
		}
	}
	return false
}

// BoxVisible reports whether any tile of target falls within radius of any
// watcher. Uses AABB overlap: expand each watcher by radius and intersect.
func BoxVisible(ws []Watcher, target Watcher, radius int) bool {
	tx2, ty2 := target.X+target.W-1, target.Y+target.H-1
	for _, w := range ws {
		if w.X-radius <= tx2 && w.X+w.W-1+radius >= target.X &&
			w.Y-radius <= ty2 && w.Y+w.H-1+radius >= target.Y {
			return true
		}
	}
	return false
}

// CityVisible reports whether any tile of target is visible.
func CityVisible(ws []Watcher, target City, radius int) bool {
	return BoxVisible(ws, CityWatcher(target), radius)
}

// FilterCities returns the subset of all visible to the given watchers.
func FilterCities(ws []Watcher, all []City, radius int) []City {
	out := make([]City, 0, len(all))
	for _, c := range all {
		if CityVisible(ws, c, radius) {
			out = append(out, c)
		}
	}
	return out
}

// FilterBuildings returns the subset of buildings visible to the given watchers.
func FilterBuildings(ws []Watcher, all []Building, radius int) []Building {
	out := make([]Building, 0, len(all))
	for _, b := range all {
		if PointVisible(ws, b.X, b.Y, radius) {
			out = append(out, b)
		}
	}
	return out
}

// FilterArmies returns the subset of armies visible to the given watchers.
func FilterArmies(ws []Watcher, all []Army, radius int) []Army {
	out := make([]Army, 0, len(all))
	for _, a := range all {
		if PointVisible(ws, a.X, a.Y, radius) {
			out = append(out, a)
		}
	}
	return out
}

// EachVisibleTile calls fn for every in-bounds tile within radius of any
// watcher, possibly more than once where watchers overlap. Walks each watcher's
// own neighbourhood rather than scanning the map, so the cost is proportional
// to what a player can see rather than to the size of the world.
func EachVisibleTile(ws []Watcher, radius, width, height int, fn func(x, y int)) {
	for _, w := range ws {
		for y := w.Y - radius; y < w.Y+w.H+radius; y++ {
			for x := w.X - radius; x < w.X+w.W+radius; x++ {
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
	return CityWatcher(c).DistanceTo(px, py)
}
