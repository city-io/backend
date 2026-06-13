package messages

import "cityio/internal/domain"

type UpdateTileOwnerMessage struct {
	Owner *string
}
type UpdateTileCityMessage struct {
	CityID string
}
type UpdateTileBuildingMessage struct {
	BuildingID *string
}

type GetTileMessage struct{}
type GetTileResponseMessage struct {
	City *domain.City
}
