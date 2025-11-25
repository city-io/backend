package messages

import "cityio/internal/models"

type UpdateTileOwnerMessage struct {
	Owner string
}
type UpdateTileCityMessage struct {
	CityID string
}
type UpdateTileBuildingMessage struct {
	BuildingID string
}

type GetTileMessage struct{}
type GetTileResponseMessage struct {
	City *models.City
}
