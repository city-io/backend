package messages

import "cityio/internal/domain"

type UpdateTileCityMessage struct {
	CityID string
}
type UpdateTileBuildingMessage struct {
	BuildingID *string
}

// ReconcileTilesMessage asks an entity to re-emit its authoritative tile-index
// updates, repairing any drift in the derived tile occupancy index.
type ReconcileTilesMessage struct{}

type GetTileMessage struct{}
type GetTileResponseMessage struct {
	City *domain.City
}
