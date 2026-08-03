package mapping

import (
	"sort"

	"cityio/internal/domain"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/world"
)

// VisibleIndices returns the row-major indices the watchers can currently see,
// deduplicated and sorted. Watchers overlap constantly — a city and the army
// beside it cover much the same ground — so hits are collapsed before use.
func VisibleIndices(gameWorld *world.World, ws []domain.Watcher) []int32 {
	if gameWorld == nil {
		return nil
	}
	seen := make([]bool, gameWorld.Width*gameWorld.Height)
	indices := make([]int32, 0, 256)
	domain.EachVisibleTile(ws, gameWorld.Width, gameWorld.Height, func(x, y int) {
		i := y*gameWorld.Width + x
		if !seen[i] {
			seen[i] = true
			indices = append(indices, int32(i))
		}
	})
	sort.Slice(indices, func(a, b int) bool { return indices[a] < indices[b] })
	return indices
}

// TerrainRevealFor packs the given tile indices with their static planes, for a
// TerrainReveal. The caller supplies the indices (typically the tiles a player
// just charted for the first time).
func TerrainRevealFor(gameWorld *world.World, indices []int32) *servicev1.TerrainReveal {
	if gameWorld == nil || len(indices) == 0 {
		return nil
	}
	out := &servicev1.TerrainReveal{
		Indices: indices,
		Terrain: make([]byte, len(indices)),
		Relief:  make([]byte, len(indices)),
		Feature: make([]byte, len(indices)),
		Special: make([]byte, len(indices)),
		Rivers:  make([]byte, len(indices)),
	}
	for k, idx := range indices {
		out.Terrain[k] = gameWorld.Terrain[idx]
		out.Relief[k] = gameWorld.Relief[idx]
		out.Feature[k] = gameWorld.Feature[idx]
		out.Special[k] = gameWorld.Special[idx]
		out.Rivers[k] = gameWorld.Rivers[idx]
	}
	return out
}

// ExploredPlanes packs the full static planes masked to what a player has
// charted: every unexplored tile is zeroed in every plane, so nothing about
// unseen ground leaks. Used to bootstrap a client.
func ExploredPlanes(gameWorld *world.World, explored domain.Explored) *servicev1.GetTerrainResponse {
	n := gameWorld.Width * gameWorld.Height
	resp := &servicev1.GetTerrainResponse{
		Width:    int32(gameWorld.Width),
		Height:   int32(gameWorld.Height),
		Seed:     gameWorld.Seed,
		Terrain:  make([]byte, n),
		Relief:   make([]byte, n),
		Feature:  make([]byte, n),
		Special:  make([]byte, n),
		Rivers:   make([]byte, n),
		Explored: []byte(explored),
	}
	for i := 0; i < n; i++ {
		if !explored.Has(i) {
			continue
		}
		resp.Terrain[i] = gameWorld.Terrain[i]
		resp.Relief[i] = gameWorld.Relief[i]
		resp.Feature[i] = gameWorld.Feature[i]
		resp.Special[i] = gameWorld.Special[i]
		resp.Rivers[i] = gameWorld.Rivers[i]
	}
	return resp
}

// SameIndices reports whether two ascending index lists are identical.
func SameIndices(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
