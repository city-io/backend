package messages

type UpdateTileCityMessage struct {
	CityID string
}
type UpdateTileBuildingMessage struct {
	BuildingID *string
}

// AddTileArmyMessage / RemoveTileArmyMessage maintain the set of armies present
// on a tile. Armies stack, so a tile can hold several.
type AddTileArmyMessage struct {
	ArmyID string
}
type RemoveTileArmyMessage struct {
	ArmyID string
}

// ReconcileTilesMessage asks an entity to re-emit its authoritative tile-index
// updates, repairing any drift in the derived tile occupancy index.
type ReconcileTilesMessage struct{}

type GetTileMessage struct{}
type GetTileResponseMessage struct {
	CityID     *string
	BuildingID *string
	ArmyIDs    []string
}
