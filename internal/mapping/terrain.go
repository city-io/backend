package mapping

import (
	"sort"

	"cityio/internal/domain"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/world"
)

// VisibleTerrainToProto packs the ground currently visible to ws.
//
// Watchers overlap constantly — a city and the army garrisoned beside it cover
// much the same tiles — so hits are deduplicated before packing. Indices are
// sorted so two calls with the same vision produce byte-identical output, which
// is what lets the stream cheaply decide whether anything actually changed.
func VisibleTerrainToProto(gameWorld *world.World, ws []domain.Watcher, radius int) *servicev1.VisibleTerrain {
	if gameWorld == nil {
		return nil
	}

	seen := make([]bool, gameWorld.Width*gameWorld.Height)
	indices := make([]int32, 0, 256)
	domain.EachVisibleTile(ws, radius, gameWorld.Width, gameWorld.Height, func(x, y int) {
		i := y*gameWorld.Width + x
		if seen[i] {
			return
		}
		seen[i] = true
		indices = append(indices, int32(i))
	})
	sort.Slice(indices, func(a, b int) bool { return indices[a] < indices[b] })

	out := &servicev1.VisibleTerrain{
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

// SameVisibleTerrain reports whether two packed sets cover the same tiles.
// Only the indices are compared: terrain itself never changes, so identical
// coverage means identical content.
func SameVisibleTerrain(a, b *servicev1.VisibleTerrain) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if len(a.GetIndices()) != len(b.GetIndices()) {
		return false
	}
	for i, v := range a.GetIndices() {
		if b.GetIndices()[i] != v {
			return false
		}
	}
	return true
}
