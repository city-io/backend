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
