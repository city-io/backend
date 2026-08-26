package actors

import (
	"log/slog"

	"cityio/internal/constants"
	"cityio/internal/domain"
)

func (state *baseActor) recordExploration(userID string, vision domain.Vision) {
	tiles := vision.VisibleCoordinates(constants.MapSize, constants.MapSize, constants.VisionRadius)
	if err := state.Store.AddExploredTiles(state.Ctx(), userID, tiles); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to persist explored tiles", "user_id", userID, "error", err)
	}
}

func (state *baseActor) structureVisionPoints(userID string) []domain.VisionPoint {
	buildings, err := state.Store.GetBuildingsByOwner(state.Ctx(), userID)
	if err != nil {
		return nil
	}
	points := make([]domain.VisionPoint, 0, len(buildings))
	for _, building := range buildings {
		if !constants.IsStandaloneStructure(building.BuildingType()) {
			continue
		}
		points = append(points, domain.VisionPoint{
			X: building.X, Y: building.Y,
			Radius: constants.GetBuildingVisionRadius(building.BuildingType(), building.Level),
		})
	}
	return points
}
