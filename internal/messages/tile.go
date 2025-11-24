package messages

import "cityio/internal/models"

type CreateTileMessage struct {
	Tile    models.Tile
	Restore bool
}
type CreateTileResponseMessage struct{}

type UpdateTileCityMessage struct {
	CityID string
}
type UpdateTileBuildingMessage struct {
	BuildingID string
}

type GetTileMessage struct{}
type GetTileResponseMessage struct {
	Tile models.Tile
	City *models.City
}

// type AddTileArmyMessage struct {
// 	ArmyPID *actor.PID
// 	Army    models.Army
// }
// type RemoveTileArmyMessage struct {
// 	Owner  string
// 	ArmyId string
// }
// type GetMapTileArmiesMessage struct{}

// type AddCityToTileResponseMessage struct {
// 	Error error
// }
// type AddBuildingToTileResponseMessage struct {
// 	Error error
// }
// type AddTileArmyResponseMessage struct {
// 	Error error
// }
// type RemoveTileArmyResponseMessage struct {
// 	Error error
// }
// type GetMapTileArmiesResponseMessage struct {
// 	Armies map[string][]*models.Army
// }

// // Errors
// type MapTileNotFoundError struct {
// 	X int
// 	Y int
// }

// func (e *MapTileNotFoundError) Error() string {
// 	return fmt.Sprintf("Map tile not found: %d,%d", e.X, e.Y)
// }
