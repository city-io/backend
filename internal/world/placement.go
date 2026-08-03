package world

// Coastal reports whether a tile touches water.
func (w *World) Coastal(x, y int) bool {
	found := false
	w.forEachNeighbor(x, y, func(j, _, _, _ int) {
		if isWater(w.Terrain[j]) {
			found = true
		}
	})
	return found
}

// HasRiver reports whether a river runs through a tile.
func (w *World) HasRiver(x, y int) bool {
	return w.InBounds(x, y) && w.Rivers[w.Index(x, y)] != 0
}

// BlockBuildable reports whether every tile of a size x size block can be
// settled.
func (w *World) BlockBuildable(x, y, size int) bool {
	for dx := 0; dx < size; dx++ {
		for dy := 0; dy < size; dy++ {
			if !w.Buildable(x+dx, y+dy) {
				return false
			}
		}
	}
	return true
}

// FindStart searches the whole map for the best place to seat a new player.
//
// Drawing random empty blocks and hoping one is habitable does not work: towns
// are seeded first and take the good land, so whatever is left unoccupied is
// disproportionately the water and mountain the town seeder rejected. A player
// picked that way reliably lands in a mountain range. Scanning is cheap — a few
// thousand blocks — so search properly instead.
//
// occupied reports whether a block of the given size at (x, y) collides with
// anything already placed. pick chooses among the shortlist.
func (w *World) FindStart(size int, occupied func(x, y int) bool, pick func(n int) int) (int, int, bool) {
	type candidate struct{ x, y, score int }
	candidates := make([]candidate, 0, 256)
	best := 0

	for y := 0; y+size <= w.Height; y++ {
		for x := 0; x+size <= w.Width; x++ {
			if occupied(x, y) {
				continue
			}
			score := w.StartScore(x, y, size)
			if score <= 0 {
				continue
			}
			if score > best {
				best = score
			}
			candidates = append(candidates, candidate{x, y, score})
		}
	}
	if len(candidates) == 0 {
		return 0, 0, false
	}

	// Shortlist everything close to the best rather than the single optimum,
	// so consecutive registrations don't all land on the same tile.
	threshold := best * 85 / 100
	shortlist := candidates[:0]
	for _, c := range candidates {
		if c.score >= threshold {
			shortlist = append(shortlist, c)
		}
	}
	c := shortlist[pick(len(shortlist))]
	return c.x, c.y, true
}

// StartScore rates a block as a starting location for a new player. A block
// with any unbuildable tile scores zero, so nobody is seated half in the sea or
// astride a mountain range; beyond that it rewards the things that make a Civ
// start worth having — fertile ground, fresh water and a coastline.
func (w *World) StartScore(x, y, size int) int {
	if !w.BlockBuildable(x, y, size) {
		return 0
	}
	score := 1
	coastal, river := false, false
	for dx := 0; dx < size; dx++ {
		for dy := 0; dy < size; dy++ {
			cx, cy := x+dx, y+dy
			g, r, f := w.TerrainAt(cx, cy)
			switch g {
			case Grassland:
				score += 3
			case Plains, Beach:
				score += 2
			case Desert, Tundra:
				score--
			}
			if r == Hills {
				score++
			}
			switch f {
			case Forest:
				score++
			case Marsh:
				score -= 2
			}
			if w.HasRiver(cx, cy) {
				river = true
			}
			if w.Coastal(cx, cy) {
				coastal = true
			}
		}
	}
	if river {
		score += 8
	}
	if coastal {
		score += 5
	}
	return score
}
