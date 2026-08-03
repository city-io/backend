package domain

// Explored is the set of tiles a player has ever seen, as a bitset in row-major
// order: tile (x, y) is bit y*width + x.
//
// Terrain is remembered once seen — you learn a tile is mountain or ocean and
// keep that knowledge — while cities, buildings and armies on it need live
// vision, because those move and change. This split is what makes scouting
// worth doing: you chart the land permanently, but you only know what is
// standing on it while someone is looking.
type Explored []byte

// NewExplored returns a bitset sized for the map, preserving any bits already
// set. Stored values begin as an empty bytea, so callers size them on load.
func NewExplored(existing []byte, width, height int) Explored {
	size := (width*height + 7) / 8
	if len(existing) >= size {
		return Explored(existing)
	}
	grown := make([]byte, size)
	copy(grown, existing)
	return Explored(grown)
}

// Has reports whether tile index i has been seen.
func (e Explored) Has(i int) bool {
	b := i >> 3
	return b >= 0 && b < len(e) && e[b]&(1<<uint(i&7)) != 0
}

// set marks index i, reporting whether it was newly set.
func (e Explored) set(i int) bool {
	b := i >> 3
	if b < 0 || b >= len(e) {
		return false
	}
	mask := byte(1) << uint(i&7)
	if e[b]&mask != 0 {
		return false
	}
	e[b] |= mask
	return true
}

// Reveal marks every tile currently visible to ws and returns the indices that
// were newly discovered. An empty result means the player's charted area has
// not grown — the common case once they stop pushing into new ground. Order is
// unspecified: the client applies reveals by index, so it does not matter.
func (e Explored) Reveal(ws []Watcher, width, height int) []int32 {
	var revealed []int32
	EachVisibleTile(ws, width, height, func(x, y int) {
		if e.set(y*width + x) {
			revealed = append(revealed, int32(y*width+x))
		}
	})
	return revealed
}
