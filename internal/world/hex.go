package world

import "math"

// The map is a flat-top hex grid in odd-q offset coordinates, squashed
// vertically by iso for the client's 2.5D projection. These constants must stay
// in step with web/src/lib/game/hex.ts: the server decides where rivers run and
// which tiles border which, and the client draws that geometry.
const iso = 0.5

var rowPitch = math.Sqrt(3) * iso

// hexPoint returns a tile centre in unit-hex space, where one hex is 2.0 wide.
//
// Noise is sampled here rather than directly at (col, row). Columns sit 1.5
// apart while rows sit only rowPitch (~0.866) apart, so sampling in grid space
// stretches every coastline and mountain range by about 1.73x horizontally once
// the map is drawn.
func hexPoint(col, row int) (float64, float64) {
	return float64(col) * 1.5, (float64(row) + 0.5*float64(col&1)) * rowPitch
}

// neighborOffsets holds the six neighbour deltas for odd-q offset coordinates,
// indexed by column parity then by direction 0..5. Direction i shares the edge
// running from vertex i to vertex (i+1)%6, which makes the reciprocal of i
// exactly (i+3)%6 — the property river links rely on to join without a seam.
var neighborOffsets = [2][6][2]int{
	{{1, 0}, {0, 1}, {-1, 0}, {-1, -1}, {0, -1}, {1, -1}},
	{{1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {0, -1}, {1, 0}},
}

func opposite(dir int) int { return (dir + 3) % 6 }

// forEachNeighbor calls fn for each in-bounds neighbour of (x, y).
func (w *World) forEachNeighbor(x, y int, fn func(j, dir, nx, ny int)) {
	offs := &neighborOffsets[x&1]
	for dir := 0; dir < 6; dir++ {
		nx, ny := x+offs[dir][0], y+offs[dir][1]
		if nx < 0 || ny < 0 || nx >= w.Width || ny >= w.Height {
			continue
		}
		fn(ny*w.Width+nx, dir, nx, ny)
	}
}
