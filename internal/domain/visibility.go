package domain

// PointVisible reports whether (px, py) is within Chebyshev distance radius
// of any tile belonging to any of the given cities.
func PointVisible(cities []City, px, py, radius int) bool {
	for i := range cities {
		c := &cities[i]
		dx := max(0, c.StartX-px, px-(c.StartX+c.Size-1))
		dy := max(0, c.StartY-py, py-(c.StartY+c.Size-1))
		if max(dx, dy) <= radius {
			return true
		}
	}
	return false
}

// CityVisible reports whether any tile of target falls within Chebyshev
// distance radius of any tile in any of the given cities. Uses AABB overlap:
// expand each city box by radius and check intersection with the target box.
func CityVisible(cities []City, target City, radius int) bool {
	tx1, ty1 := target.StartX, target.StartY
	tx2, ty2 := target.StartX+target.Size-1, target.StartY+target.Size-1
	for i := range cities {
		c := &cities[i]
		ox1 := c.StartX - radius
		oy1 := c.StartY - radius
		ox2 := c.StartX + c.Size - 1 + radius
		oy2 := c.StartY + c.Size - 1 + radius
		if ox1 <= tx2 && ox2 >= tx1 && oy1 <= ty2 && oy2 >= ty1 {
			return true
		}
	}
	return false
}

// FilterCities returns the subset of all visible from cities within radius.
func FilterCities(cities, all []City, radius int) []City {
	out := make([]City, 0, len(all))
	for _, c := range all {
		if CityVisible(cities, c, radius) {
			out = append(out, c)
		}
	}
	return out
}

// FilterBuildings returns the subset of buildings visible from cities within radius.
func FilterBuildings(cities []City, all []Building, radius int) []Building {
	out := make([]Building, 0, len(all))
	for _, b := range all {
		if PointVisible(cities, b.X, b.Y, radius) {
			out = append(out, b)
		}
	}
	return out
}
