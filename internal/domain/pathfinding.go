package domain

import "container/heap"

type pathNode struct {
	coords Coordinates
	cost   int
	score  int
}

type pathQueue []*pathNode

func (q pathQueue) Len() int { return len(q) }
func (q pathQueue) Less(i, j int) bool {
	return q[i].score < q[j].score || (q[i].score == q[j].score && q[i].cost < q[j].cost)
}
func (q pathQueue) Swap(i, j int)   { q[i], q[j] = q[j], q[i] }
func (q *pathQueue) Push(value any) { *q = append(*q, value.(*pathNode)) }
func (q *pathQueue) Pop() any {
	old := *q
	last := old[len(old)-1]
	*q = old[:len(old)-1]
	return last
}

var pathDirections = [...]Coordinates{
	{X: -1, Y: -1}, {X: 0, Y: -1}, {X: 1, Y: -1},
	{X: -1, Y: 0}, {X: 1, Y: 0},
	{X: -1, Y: 1}, {X: 0, Y: 1}, {X: 1, Y: 1},
}

// TerrainMovementCost returns a tile's movement-time multiplier. Zero means
// current land armies cannot enter it.
func TerrainMovementCost(terrain TerrainType) int {
	switch terrain {
	case TerrainTypeWater:
		return 0
	case TerrainTypeMarsh:
		return 2
	case TerrainTypeMountains:
		return 3
	default:
		return 1
	}
}

// FindLandPath returns a lowest-cost 8-direction route excluding the start tile.
func FindLandPath(grid TerrainGrid, start, destination Coordinates) ([]Coordinates, bool) {
	if start == destination {
		return []Coordinates{}, true
	}
	if !traversable(grid, start) || !traversable(grid, destination) {
		return nil, false
	}

	frontier := pathQueue{&pathNode{coords: start, score: chebyshev(start, destination)}}
	heap.Init(&frontier)
	costs := map[Coordinates]int{start: 0}
	previous := make(map[Coordinates]Coordinates)

	for frontier.Len() > 0 {
		current := heap.Pop(&frontier).(*pathNode)
		if known := costs[current.coords]; current.cost != known {
			continue
		}
		if current.coords == destination {
			return buildPath(previous, start, destination), true
		}
		for _, direction := range pathDirections {
			next := Coordinates{X: current.coords.X + direction.X, Y: current.coords.Y + direction.Y}
			terrain, ok := grid.At(next.X, next.Y)
			if !ok || TerrainMovementCost(terrain) == 0 || cutsBlockedCorner(grid, current.coords, direction) {
				continue
			}
			cost := current.cost + TerrainMovementCost(terrain)
			known, seen := costs[next]
			if seen && cost >= known {
				continue
			}
			costs[next] = cost
			previous[next] = current.coords
			heap.Push(&frontier, &pathNode{coords: next, cost: cost, score: cost + chebyshev(next, destination)})
		}
	}
	return nil, false
}

// FindKnownLandPath plans with explored terrain while treating undiscovered
// tiles as ordinary traversable land. If a discovered impassable destination
// cannot be reached, it returns a route to the closest reachable explored land
// and reports reachesDestination=false.
func FindKnownLandPath(grid TerrainGrid, explored map[Coordinates]struct{}, start, destination Coordinates) ([]Coordinates, bool) {
	masked := TerrainGrid{Width: grid.Width, Height: grid.Height, Tiles: make([]TerrainType, len(grid.Tiles))}
	for y := 0; y < grid.Height; y++ {
		for x := 0; x < grid.Width; x++ {
			coords := Coordinates{X: x, Y: y}
			if _, known := explored[coords]; known || coords == start {
				masked.Tiles[y*grid.Width+x] = grid.Tiles[y*grid.Width+x]
			} else {
				masked.Tiles[y*grid.Width+x] = TerrainTypePlains
			}
		}
	}

	if path, ok := FindLandPath(masked, start, destination); ok {
		return path, true
	}

	var best Coordinates
	found := false
	bestDistance := int(^uint(0) >> 1)
	bestCost := bestDistance
	frontier := pathQueue{&pathNode{coords: start}}
	heap.Init(&frontier)
	costs := map[Coordinates]int{start: 0}
	previous := make(map[Coordinates]Coordinates)
	for frontier.Len() > 0 {
		current := heap.Pop(&frontier).(*pathNode)
		if current.cost != costs[current.coords] {
			continue
		}
		if _, isExplored := explored[current.coords]; isExplored {
			distance := chebyshev(current.coords, destination)
			if distance < bestDistance || (distance == bestDistance && current.cost < bestCost) {
				best, bestDistance, bestCost, found = current.coords, distance, current.cost, true
			}
		}
		for _, direction := range pathDirections {
			next := Coordinates{X: current.coords.X + direction.X, Y: current.coords.Y + direction.Y}
			terrain, ok := masked.At(next.X, next.Y)
			if !ok || TerrainMovementCost(terrain) == 0 || cutsBlockedCorner(masked, current.coords, direction) {
				continue
			}
			cost := current.cost + TerrainMovementCost(terrain)
			if known, seen := costs[next]; seen && cost >= known {
				continue
			}
			costs[next] = cost
			previous[next] = current.coords
			heap.Push(&frontier, &pathNode{coords: next, cost: cost, score: cost})
		}
	}
	if !found {
		return []Coordinates{}, false
	}
	return buildPath(previous, start, best), false
}

func traversable(grid TerrainGrid, coords Coordinates) bool {
	terrain, ok := grid.At(coords.X, coords.Y)
	return ok && TerrainMovementCost(terrain) > 0
}

func cutsBlockedCorner(grid TerrainGrid, current, direction Coordinates) bool {
	if direction.X == 0 || direction.Y == 0 {
		return false
	}
	return !traversable(grid, Coordinates{X: current.X + direction.X, Y: current.Y}) ||
		!traversable(grid, Coordinates{X: current.X, Y: current.Y + direction.Y})
}

func buildPath(previous map[Coordinates]Coordinates, start, destination Coordinates) []Coordinates {
	path := make([]Coordinates, 0)
	for current := destination; current != start; current = previous[current] {
		path = append(path, current)
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func chebyshev(a, b Coordinates) int {
	return max(abs(a.X-b.X), abs(a.Y-b.Y))
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
