package domain

type VisionPoint struct {
	X      int
	Y      int
	Radius int
}

// Vision contains the settlements, armies, and structures that reveal the map.
type Vision struct {
	Cities []City
	Armies []Army
	Points []VisionPoint
}

// VisibleCoordinates returns every in-bounds tile currently revealed by this
// vision. The result is row-major and contains no duplicates.
func (v Vision) VisibleCoordinates(width, height, radius int) []Coordinates {
	visible := make([]bool, width*height)
	mark := func(minX, minY, maxX, maxY int) {
		minX, minY = max(minX, 0), max(minY, 0)
		maxX, maxY = min(maxX, width-1), min(maxY, height-1)
		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				visible[y*width+x] = true
			}
		}
	}
	for _, city := range v.Cities {
		mark(city.StartX-radius, city.StartY-radius, city.StartX+city.Size-1+radius, city.StartY+city.Size-1+radius)
	}
	for _, army := range v.Armies {
		mark(army.X-radius, army.Y-radius, army.X+radius, army.Y+radius)
	}
	for _, point := range v.Points {
		mark(point.X-point.Radius, point.Y-point.Radius, point.X+point.Radius, point.Y+point.Radius)
	}
	result := make([]Coordinates, 0)
	for index, isVisible := range visible {
		if isVisible {
			result = append(result, Coordinates{X: index % width, Y: index / width})
		}
	}
	return result
}

// PointVisible reports whether (px, py) is within Chebyshev distance radius
// of any tile belonging to a city or any army position in the vision.
func (v Vision) PointVisible(px, py, radius int) bool {
	for i := range v.Cities {
		c := &v.Cities[i]
		dx := max(0, c.StartX-px, px-(c.StartX+c.Size-1))
		dy := max(0, c.StartY-py, py-(c.StartY+c.Size-1))
		if max(dx, dy) <= radius {
			return true
		}
	}
	for i := range v.Armies {
		a := &v.Armies[i]
		if max(abs(a.X-px), abs(a.Y-py)) <= radius {
			return true
		}
	}
	for _, point := range v.Points {
		if max(abs(point.X-px), abs(point.Y-py)) <= point.Radius {
			return true
		}
	}
	return false
}

// CityVisible reports whether any tile of target is visible.
func (v Vision) CityVisible(target City, radius int) bool {
	tx1, ty1 := target.StartX, target.StartY
	tx2, ty2 := target.StartX+target.Size-1, target.StartY+target.Size-1
	for i := range v.Cities {
		c := &v.Cities[i]
		ox1 := c.StartX - radius
		oy1 := c.StartY - radius
		ox2 := c.StartX + c.Size - 1 + radius
		oy2 := c.StartY + c.Size - 1 + radius
		if ox1 <= tx2 && ox2 >= tx1 && oy1 <= ty2 && oy2 >= ty1 {
			return true
		}
	}
	for i := range v.Armies {
		a := &v.Armies[i]
		if a.X+radius >= tx1 && a.X-radius <= tx2 && a.Y+radius >= ty1 && a.Y-radius <= ty2 {
			return true
		}
	}
	for _, point := range v.Points {
		if point.X+point.Radius >= tx1 && point.X-point.Radius <= tx2 && point.Y+point.Radius >= ty1 && point.Y-point.Radius <= ty2 {
			return true
		}
	}
	return false
}

// FilterCities returns the subset of all visible within radius.
func (v Vision) FilterCities(all []City, radius int) []City {
	out := make([]City, 0, len(all))
	for _, c := range all {
		if v.CityVisible(c, radius) {
			out = append(out, c)
		}
	}
	return out
}

// FilterBuildings returns the subset of buildings visible within radius.
func (v Vision) FilterBuildings(all []Building, radius int) []Building {
	out := make([]Building, 0, len(all))
	for _, b := range all {
		if v.PointVisible(b.X, b.Y, radius) {
			out = append(out, b)
		}
	}
	return out
}

// FilterArmies returns the subset of armies visible within radius.
func (v Vision) FilterArmies(all []Army, radius int) []Army {
	out := make([]Army, 0, len(all))
	for _, a := range all {
		if v.PointVisible(a.X, a.Y, radius) {
			out = append(out, a)
		}
	}
	return out
}

// ChebyshevToCity returns the Chebyshev (king-move) distance from point (px, py)
// to the nearest tile of city c. Zero when the point lies inside the city box.
func ChebyshevToCity(c City, px, py int) int {
	dx := max(0, c.StartX-px, px-(c.StartX+c.Size-1))
	dy := max(0, c.StartY-py, py-(c.StartY+c.Size-1))
	return max(dx, dy)
}
