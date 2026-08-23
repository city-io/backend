// Package mapping converts between pure domain entities and the generated
// protobuf transport types. It is the only place that knows about both, so the
// domain package stays free of transport concerns.
package mapping

import (
	"sort"

	"google.golang.org/protobuf/types/known/timestamppb"

	entityv1 "cityio/internal/gen/cityio/entity/v1"
	servicev1 "cityio/internal/gen/cityio/service/v1"

	"cityio/internal/constants"
	"cityio/internal/domain"
)

var cityTypeToProto = map[domain.CityType]entityv1.CityType{
	domain.CityTypeCity: entityv1.CityType_CITY_TYPE_CITY,
	domain.CityTypeTown: entityv1.CityType_CITY_TYPE_TOWN,
}

var cityTypeFromProto = map[entityv1.CityType]domain.CityType{
	entityv1.CityType_CITY_TYPE_CITY: domain.CityTypeCity,
	entityv1.CityType_CITY_TYPE_TOWN: domain.CityTypeTown,
}

var buildingTypeToProto = map[domain.BuildingType]entityv1.BuildingType{
	domain.BuildingTypeCityCenter: entityv1.BuildingType_BUILDING_TYPE_CITY_CENTER,
	domain.BuildingTypeTownCenter: entityv1.BuildingType_BUILDING_TYPE_TOWN_CENTER,
	domain.BuildingTypeBarracks:   entityv1.BuildingType_BUILDING_TYPE_BARRACKS,
	domain.BuildingTypeHouse:      entityv1.BuildingType_BUILDING_TYPE_HOUSE,
	domain.BuildingTypeFarm:       entityv1.BuildingType_BUILDING_TYPE_FARM,
	domain.BuildingTypeMine:       entityv1.BuildingType_BUILDING_TYPE_MINE,
}

var buildingTypeFromProto = map[entityv1.BuildingType]domain.BuildingType{
	entityv1.BuildingType_BUILDING_TYPE_CITY_CENTER: domain.BuildingTypeCityCenter,
	entityv1.BuildingType_BUILDING_TYPE_TOWN_CENTER: domain.BuildingTypeTownCenter,
	entityv1.BuildingType_BUILDING_TYPE_BARRACKS:    domain.BuildingTypeBarracks,
	entityv1.BuildingType_BUILDING_TYPE_HOUSE:       domain.BuildingTypeHouse,
	entityv1.BuildingType_BUILDING_TYPE_FARM:        domain.BuildingTypeFarm,
	entityv1.BuildingType_BUILDING_TYPE_MINE:        domain.BuildingTypeMine,
}

var terrainTypeToProto = map[domain.TerrainType]entityv1.TerrainType{
	domain.TerrainTypeGrassland: entityv1.TerrainType_TERRAIN_TYPE_GRASSLAND,
	domain.TerrainTypePlains:    entityv1.TerrainType_TERRAIN_TYPE_PLAINS,
	domain.TerrainTypeForest:    entityv1.TerrainType_TERRAIN_TYPE_FOREST,
	domain.TerrainTypeHills:     entityv1.TerrainType_TERRAIN_TYPE_HILLS,
	domain.TerrainTypeMountains: entityv1.TerrainType_TERRAIN_TYPE_MOUNTAINS,
	domain.TerrainTypeDesert:    entityv1.TerrainType_TERRAIN_TYPE_DESERT,
	domain.TerrainTypeMarsh:     entityv1.TerrainType_TERRAIN_TYPE_MARSH,
	domain.TerrainTypeWater:     entityv1.TerrainType_TERRAIN_TYPE_WATER,
}

var troopTypeToProto = map[domain.TroopType]entityv1.TroopType{
	domain.TroopTypeSoldier:   entityv1.TroopType_TROOP_TYPE_SOLDIER,
	domain.TroopTypeArcher:    entityv1.TroopType_TROOP_TYPE_ARCHER,
	domain.TroopTypeCavalry:   entityv1.TroopType_TROOP_TYPE_CAVALRY,
	domain.TroopTypeArtillery: entityv1.TroopType_TROOP_TYPE_ARTILLERY,
}

var troopTypeFromProto = map[entityv1.TroopType]domain.TroopType{
	entityv1.TroopType_TROOP_TYPE_SOLDIER:   domain.TroopTypeSoldier,
	entityv1.TroopType_TROOP_TYPE_ARCHER:    domain.TroopTypeArcher,
	entityv1.TroopType_TROOP_TYPE_CAVALRY:   domain.TroopTypeCavalry,
	entityv1.TroopType_TROOP_TYPE_ARTILLERY: domain.TroopTypeArtillery,
}

// ToUserId wraps a raw string into a typed proto ID.
func ToUserId(id string) *entityv1.UserId {
	return &entityv1.UserId{Value: id}
}

// ToCityId wraps a raw string into a typed proto ID.
func ToCityId(id string) *entityv1.CityId {
	return &entityv1.CityId{Value: id}
}

// ToBuildingId wraps a raw string into a typed proto ID.
func ToBuildingId(id string) *entityv1.BuildingId {
	return &entityv1.BuildingId{Value: id}
}

// ToArmyId wraps a raw string into a typed proto ID.
func ToArmyId(id string) *entityv1.ArmyId {
	return &entityv1.ArmyId{Value: id}
}

func ToArmyOrderId(id string) *entityv1.ArmyOrderId {
	return &entityv1.ArmyOrderId{Value: id}
}

func ToBattleId(id string) *entityv1.BattleId {
	return &entityv1.BattleId{Value: id}
}

func ToMailboxMessageId(id string) *entityv1.MailboxMessageId {
	return &entityv1.MailboxMessageId{Value: id}
}

// ToTrainingOrderId wraps a raw string into a typed proto ID.
func ToTrainingOrderId(id string) *entityv1.TrainingOrderId {
	return &entityv1.TrainingOrderId{Value: id}
}

// ToTileId identifies a tile by its immutable map coordinates.
func ToTileId(x, y int) *entityv1.TileId {
	return &entityv1.TileId{X: int32(x), Y: int32(y)}
}

// TroopTypeToProto maps a domain troop type to its proto enum.
func TroopTypeToProto(t domain.TroopType) entityv1.TroopType {
	return troopTypeToProto[t]
}

// TroopTypeFromProto maps a proto troop type enum to its domain value.
func TroopTypeFromProto(t entityv1.TroopType) domain.TroopType {
	return troopTypeFromProto[t]
}

// TrainingOrderToProto converts a queued training order to its API shape.
func TrainingOrderToProto(order domain.TrainingOrder) *servicev1.TrainingOrder {
	result := &servicev1.TrainingOrder{
		TrainingOrderId: ToTrainingOrderId(order.TrainingOrderID),
		ArmyId:          ToArmyId(order.ArmyID),
		BarracksId:      ToBuildingId(order.BarracksID),
		Type:            TroopTypeToProto(order.TroopType),
		Count:           int32(order.Count),
	}
	if order.StartedAt.Time != nil {
		result.StartedAt = timestamppb.New(*order.StartedAt.Time)
	}
	if order.CompletesAt.Time != nil {
		result.CompletesAt = timestamppb.New(*order.CompletesAt.Time)
	}
	return result
}

// CityTypeToProto maps a domain city type to its proto enum.
func CityTypeToProto(t domain.CityType) entityv1.CityType {
	return cityTypeToProto[t]
}

// CityTypeFromProto maps a proto city type enum to its domain value.
func CityTypeFromProto(t entityv1.CityType) domain.CityType {
	return cityTypeFromProto[t]
}

// BuildingTypeToProto maps a domain building type to its proto enum.
func BuildingTypeToProto(t domain.BuildingType) entityv1.BuildingType {
	return buildingTypeToProto[t]
}

// BuildingTypeFromProto maps a proto building type enum to its domain value.
func BuildingTypeFromProto(t entityv1.BuildingType) domain.BuildingType {
	return buildingTypeFromProto[t]
}

// RatePerHour wraps a per-hour amount as a Rate proto with scale=3600.
func RatePerHour(perHour int64) *entityv1.Rate {
	return &entityv1.Rate{Value: perHour, Scale: 3600}
}

// UserToProto converts a domain user to its proto representation. The password
// is never copied across the wire.
func UserToProto(u domain.User) *entityv1.User {
	return &entityv1.User{
		UserId:     ToUserId(u.UserID),
		Email:      u.Email,
		Username:   u.Username,
		Gold:       u.Gold,
		Food:       u.Food,
		FoodIncome: RatePerHour(u.FoodIncomeRate),
		FoodUpkeep: RatePerHour(u.FoodUpkeepRate),
	}
}

// CityToProto converts a domain city to its proto representation.
func CityToProto(c domain.City) *entityv1.City {
	out := &entityv1.City{
		CityId:                    ToCityId(c.CityID),
		Type:                      CityTypeToProto(c.Type),
		Name:                      c.Name,
		Population:                c.Population,
		PopulationCap:             c.PopulationCap,
		Start:                     &entityv1.Coordinates{X: int32(c.StartX), Y: int32(c.StartY)},
		Size:                      int32(c.Size),
		FoodProduction:            RatePerHour(c.FoodProductionRate),
		FoodUpkeep:                RatePerHour(c.FoodUpkeep),
		NetFoodFlow:               RatePerHour(c.NetFoodFlow),
		Starving:                  c.Starving,
		PopulationGrowth:          RatePerHour(c.PopulationGrowthRate),
		MilitiaPopulation:         c.MilitiaPopulation,
		MilitiaTarget:             c.MilitiaTarget,
		MilitiaPercent:            constants.MilitiaPercent(c),
		CorePopulation:            constants.CorePopulation(c),
		CorePopulationFloor:       constants.ProtectedCorePopulation(c),
		RecruitablePopulation:     constants.RecruitablePopulationExact(c),
		TaxablePopulation:         constants.TaxablePopulation(c),
		TaxRatePercent:            int32(c.TaxRatePercent),
		TaxIncome:                 RatePerHour(constants.TaxIncomePerHour(c)),
		PopulationGrowthBeforeTax: RatePerHour(c.PopulationGrowthBeforeTaxRate),
		DemographicsVisible:       true,
	}
	if c.Owner != nil {
		out.Owner = ToUserId(*c.Owner)
	}
	return out
}

// HidePrivateCityFields leaves only a settlement's identity, ownership, type,
// and location. Exact demographic, defensive, and economic intelligence is
// owner-only until a future scouting system explicitly discloses it.
func HidePrivateCityFields(c *entityv1.City) {
	c.DemographicsVisible = false
	c.Population = 0
	c.PopulationCap = 0
	c.Starving = false
	c.PopulationGrowth = nil
	c.MilitiaPopulation = 0
	c.MilitiaTarget = 0
	c.MilitiaPercent = 0
	c.CorePopulation = 0
	c.TaxablePopulation = 0
	c.FoodProduction = nil
	c.FoodUpkeep = nil
	c.NetFoodFlow = nil
	c.TaxRatePercent = 0
	c.TaxIncome = nil
	c.PopulationGrowthBeforeTax = nil
	c.CorePopulationFloor = 0
	c.RecruitablePopulation = 0
}

// TerrainToProto converts a domain terrain type to its protobuf enum.
func TerrainToProto(terrain domain.TerrainType) entityv1.TerrainType {
	return terrainTypeToProto[terrain]
}

// TileToProto builds a proto Tile from terrain and occupancy data.
func TileToProto(cityID, buildingID *string, armyIDs []string, terrain domain.TerrainType, x, y int) *entityv1.Tile {
	t := &entityv1.Tile{TileId: ToTileId(x, y), Terrain: TerrainToProto(terrain)}
	if cityID != nil {
		t.CityId = ToCityId(*cityID)
	}
	if buildingID != nil {
		t.BuildingId = ToBuildingId(*buildingID)
	}
	for _, id := range armyIDs {
		t.ArmyIds = append(t.ArmyIds, ToArmyId(id))
	}
	return t
}

// MapTilesToProto builds the map's tile entities and their root IDs.
func MapTilesToProto(grid domain.TerrainGrid, cities []domain.City, buildings []domain.Building, armies []domain.Army) ([]*entityv1.TileId, []*entityv1.Tile) {
	cityAt, buildingAt, armiesAt := mapOccupancy(grid, cities, buildings, armies)

	tileIDs := make([]*entityv1.TileId, len(grid.Tiles))
	tiles := make([]*entityv1.Tile, len(grid.Tiles))
	for idx, terrain := range grid.Tiles {
		x, y := idx%grid.Width, idx/grid.Width
		tile := mappedTile(grid, cityAt, buildingAt, armiesAt, terrain, x, y)
		tileIDs[idx] = tile.TileId
		tiles[idx] = tile
	}
	return tileIDs, tiles
}

// MapTilesAroundPointToProto builds the tile entities revealed around a point.
func MapTilesAroundPointToProto(grid domain.TerrainGrid, x, y, radius int, cities []domain.City, buildings []domain.Building, armies []domain.Army) []*entityv1.Tile {
	cityAt, buildingAt, armiesAt := mapOccupancy(grid, cities, buildings, armies)
	tiles := make([]*entityv1.Tile, 0, (radius*2+1)*(radius*2+1))
	for ty := max(0, y-radius); ty < min(grid.Height, y+radius+1); ty++ {
		for tx := max(0, x-radius); tx < min(grid.Width, x+radius+1); tx++ {
			terrain, _ := grid.At(tx, ty)
			tiles = append(tiles, mappedTile(grid, cityAt, buildingAt, armiesAt, terrain, tx, ty))
		}
	}
	return tiles
}

// ExploredTilesToProto returns terrain for every explored coordinate. Dynamic
// occupancy is included only while the coordinate is currently visible.
func ExploredTilesToProto(grid domain.TerrainGrid, explored []domain.Coordinates, visible map[domain.Coordinates]struct{}, cities []domain.City, buildings []domain.Building, armies []domain.Army) []*entityv1.Tile {
	cityAt, buildingAt, armiesAt := mapOccupancy(grid, cities, buildings, armies)
	tiles := make([]*entityv1.Tile, 0, len(explored))
	for _, coords := range explored {
		terrain, ok := grid.At(coords.X, coords.Y)
		if !ok {
			continue
		}
		if _, ok := visible[coords]; !ok {
			tiles = append(tiles, TileToProto(nil, nil, nil, terrain, coords.X, coords.Y))
			continue
		}
		tiles = append(tiles, mappedTile(grid, cityAt, buildingAt, armiesAt, terrain, coords.X, coords.Y))
	}
	return tiles
}

func mapOccupancy(grid domain.TerrainGrid, cities []domain.City, buildings []domain.Building, armies []domain.Army) (map[int]string, map[int]string, map[int][]string) {
	cityAt := make(map[int]string)
	for _, city := range cities {
		for y := max(0, city.StartY); y < min(grid.Height, city.StartY+city.Size); y++ {
			for x := max(0, city.StartX); x < min(grid.Width, city.StartX+city.Size); x++ {
				cityAt[y*grid.Width+x] = city.CityID
			}
		}
	}

	buildingAt := make(map[int]string)
	for _, building := range buildings {
		if _, ok := grid.At(building.X, building.Y); ok {
			buildingAt[building.Y*grid.Width+building.X] = building.BuildingID
		}
	}

	armiesAt := make(map[int][]string)
	for _, army := range armies {
		if _, ok := grid.At(army.X, army.Y); ok {
			idx := army.Y*grid.Width + army.X
			armiesAt[idx] = append(armiesAt[idx], army.ArmyID)
		}
	}
	for index := range armiesAt {
		sort.Strings(armiesAt[index])
	}

	return cityAt, buildingAt, armiesAt
}

func mappedTile(grid domain.TerrainGrid, cityAt, buildingAt map[int]string, armiesAt map[int][]string, terrain domain.TerrainType, x, y int) *entityv1.Tile {
	idx := y*grid.Width + x
	var cityID, buildingID *string
	if id, ok := cityAt[idx]; ok {
		cityID = &id
	}
	if id, ok := buildingAt[idx]; ok {
		buildingID = &id
	}
	return TileToProto(cityID, buildingID, armiesAt[idx], terrain, x, y)
}

// BuildingToProto converts a domain building to its proto representation.
func BuildingToProto(b domain.Building) *entityv1.Building {
	out := &entityv1.Building{
		BuildingId:  ToBuildingId(b.BuildingID),
		CityId:      ToCityId(b.CityID),
		Type:        BuildingTypeToProto(b.BuildingType()),
		Level:       int32(b.Level),
		TargetLevel: int32(b.TargetLevel),
		Coords:      &entityv1.Coordinates{X: int32(b.X), Y: int32(b.Y)},
	}
	if b.ConstructionStart.Time != nil {
		out.ConstructionStart = timestamppb.New(*b.ConstructionStart.Time)
	}
	if b.ConstructionEnd.Time != nil {
		out.ConstructionEnd = timestamppb.New(*b.ConstructionEnd.Time)
	}
	return out
}

// ArmyToProto converts a domain army to its proto representation.
func ArmyToProto(a domain.Army) *entityv1.Army {
	out := &entityv1.Army{
		ArmyId:                ToArmyId(a.ArmyID),
		Owner:                 ToUserId(a.Owner),
		Coords:                &entityv1.Coordinates{X: int32(a.X), Y: int32(a.Y)},
		CompositionVisibility: entityv1.ArmyCompositionVisibility_ARMY_COMPOSITION_VISIBILITY_EXACT,
	}
	for _, troopType := range []domain.TroopType{
		domain.TroopTypeSoldier,
		domain.TroopTypeArcher,
		domain.TroopTypeCavalry,
		domain.TroopTypeArtillery,
	} {
		count := a.Troops[troopType]
		if count <= 0 {
			continue
		}
		protoCount := int32(count)
		out.Troops = append(out.Troops, &entityv1.TroopStack{Type: TroopTypeToProto(troopType), Count: &protoCount})
	}
	if a.OrderID != nil {
		out.OrderId = ToArmyOrderId(*a.OrderID)
	}
	if a.BattleID != nil {
		out.BattleId = ToBattleId(*a.BattleID)
	}
	return out
}

// HidePrivateArmyFields removes composition and march references the viewer is
// not authorized to inspect. The physical army remains visible on the map.
func HidePrivateArmyFields(a *entityv1.Army) {
	a.CompositionVisibility = entityv1.ArmyCompositionVisibility_ARMY_COMPOSITION_VISIBILITY_HIDDEN
	a.Troops = nil
	a.OrderId = nil
	a.BattleId = nil
}

func BattleToProto(b domain.Battle) *entityv1.Battle {
	out := &entityv1.Battle{
		BattleId:   ToBattleId(b.BattleID),
		TileId:     ToTileId(b.X, b.Y),
		Attackers:  battleSideToProto(b.Attackers),
		Defenders:  battleSideToProto(b.Defenders),
		StartedAt:  timestamppb.New(b.StartedAt),
		NextTickAt: timestamppb.New(b.NextTick),
	}
	return out
}

func battleSideToProto(side domain.BattleSide) *entityv1.BattleSide {
	out := &entityv1.BattleSide{MilitiaCount: side.MilitiaCount}
	for _, id := range side.UserIDs {
		out.UserIds = append(out.UserIds, ToUserId(id))
	}
	for _, id := range side.ArmyIDs {
		out.ArmyIds = append(out.ArmyIds, ToArmyId(id))
	}
	if side.MilitiaCityID != nil {
		out.MilitiaCityId = ToCityId(*side.MilitiaCityID)
	}
	return out
}

func MailboxMessageToProto(message domain.MailboxMessage) *entityv1.MailboxMessage {
	out := &entityv1.MailboxMessage{
		MailboxMessageId: ToMailboxMessageId(message.MailboxMessageID),
		RecipientId:      ToUserId(message.RecipientID),
		CreatedAt:        timestamppb.New(message.CreatedAt),
	}
	if message.ReadAt.Time != nil {
		out.ReadAt = timestamppb.New(*message.ReadAt.Time)
	}
	if message.BattleReport != nil {
		out.Content = &entityv1.MailboxMessage_BattleReport{BattleReport: battleReportToProto(*message.BattleReport)}
	}
	return out
}

func battleReportToProto(report domain.BattleReport) *entityv1.BattleReport {
	out := &entityv1.BattleReport{
		BattleId:   ToBattleId(report.BattleID),
		TileId:     ToTileId(report.X, report.Y),
		Role:       battleReportRoleToProto(report.Role),
		Outcome:    battleReportOutcomeToProto(report.Outcome),
		Engagement: battleReportEngagementToProto(report.Engagement),
		Resolution: battleReportResolutionToProto(report.Resolution),
		Attackers:  battleReportSideToProto(report.Attackers),
		Defenders:  battleReportSideToProto(report.Defenders),
		StartedAt:  timestamppb.New(report.StartedAt),
		EndedAt:    timestamppb.New(report.EndedAt),
	}
	for _, round := range report.Rounds {
		mapped := &entityv1.BattleReportRound{
			Number:         int32(round.Number),
			OccurredAt:     timestamppb.New(round.OccurredAt),
			AttackerPower:  round.AttackerPower,
			DefenderPower:  round.DefenderPower,
			AttackerLosses: battleReportLossesToProto(round.AttackerLosses),
			DefenderLosses: battleReportLossesToProto(round.DefenderLosses),
		}
		out.Rounds = append(out.Rounds, mapped)
	}
	return out
}

func battleReportSideToProto(side domain.BattleReportSide) *entityv1.BattleReportSide {
	out := &entityv1.BattleReportSide{
		StartingMilitia:  side.StartingMilitia,
		SurvivingMilitia: side.SurvivingMilitia,
	}
	for _, userID := range side.UserIDs {
		out.UserIds = append(out.UserIds, ToUserId(userID))
	}
	for _, commander := range side.Commanders {
		out.Commanders = append(out.Commanders, &entityv1.BattleReportCommander{
			UserId: ToUserId(commander.UserID), Username: commander.Username,
		})
	}
	for _, army := range side.Armies {
		out.Armies = append(out.Armies, &entityv1.BattleReportArmy{
			ArmyId:          ToArmyId(army.ArmyID),
			OwnerId:         ToUserId(army.OwnerID),
			StartingTroops:  troopCountsToProto(army.StartingTroops),
			SurvivingTroops: troopCountsToProto(army.SurvivingTroops),
			Retreated:       army.Retreated,
			Destroyed:       army.Destroyed,
		})
	}
	if side.MilitiaCityID != nil {
		out.MilitiaCityId = ToCityId(*side.MilitiaCityID)
	}
	if side.Settlement != nil {
		out.Settlement = &entityv1.BattleReportSettlement{
			CityId:             ToCityId(side.Settlement.CityID),
			Name:               side.Settlement.Name,
			Type:               CityTypeToProto(side.Settlement.Type),
			StartingPopulation: side.Settlement.StartingPopulation,
			EndingPopulation:   side.Settlement.EndingPopulation,
		}
		if side.Settlement.OwnerID != nil {
			out.Settlement.OwnerId = ToUserId(*side.Settlement.OwnerID)
		}
	}
	return out
}

func battleReportLossesToProto(losses []domain.BattleReportLoss) []*entityv1.BattleReportLoss {
	out := make([]*entityv1.BattleReportLoss, 0, len(losses))
	for _, loss := range losses {
		mapped := &entityv1.BattleReportLoss{Troops: troopCountsToProto(loss.Troops), Militia: loss.Militia}
		if loss.ArmyID != nil {
			mapped.ArmyId = ToArmyId(*loss.ArmyID)
		}
		if loss.MilitiaCityID != nil {
			mapped.MilitiaCityId = ToCityId(*loss.MilitiaCityID)
		}
		out = append(out, mapped)
	}
	return out
}

func troopCountsToProto(troops map[domain.TroopType]int64) []*entityv1.TroopStack {
	out := make([]*entityv1.TroopStack, 0, len(troops))
	for _, troopType := range constants.AllTroopTypes() {
		if count := troops[troopType]; count > 0 {
			mappedCount := int32(count)
			out = append(out, &entityv1.TroopStack{Type: TroopTypeToProto(troopType), Count: &mappedCount})
		}
	}
	return out
}

func battleReportRoleToProto(role domain.BattleReportRole) entityv1.BattleReportRole {
	if role == domain.BattleReportRoleAttacker {
		return entityv1.BattleReportRole_BATTLE_REPORT_ROLE_ATTACKER
	}
	return entityv1.BattleReportRole_BATTLE_REPORT_ROLE_DEFENDER
}

func battleReportOutcomeToProto(outcome domain.BattleReportOutcome) entityv1.BattleReportOutcome {
	switch outcome {
	case domain.BattleReportOutcomeVictory:
		return entityv1.BattleReportOutcome_BATTLE_REPORT_OUTCOME_VICTORY
	case domain.BattleReportOutcomeDraw:
		return entityv1.BattleReportOutcome_BATTLE_REPORT_OUTCOME_DRAW
	default:
		return entityv1.BattleReportOutcome_BATTLE_REPORT_OUTCOME_DEFEAT
	}
}

func battleReportEngagementToProto(engagement domain.BattleReportEngagement) entityv1.BattleReportEngagement {
	if engagement == domain.BattleReportEngagementSiege {
		return entityv1.BattleReportEngagement_BATTLE_REPORT_ENGAGEMENT_SETTLEMENT_SIEGE
	}
	return entityv1.BattleReportEngagement_BATTLE_REPORT_ENGAGEMENT_FIELD_BATTLE
}

func battleReportResolutionToProto(resolution domain.BattleReportResolution) entityv1.BattleReportResolution {
	switch resolution {
	case domain.BattleReportResolutionRetreat:
		return entityv1.BattleReportResolution_BATTLE_REPORT_RESOLUTION_RETREAT
	case domain.BattleReportResolutionMutualDestruction:
		return entityv1.BattleReportResolution_BATTLE_REPORT_RESOLUTION_MUTUAL_DESTRUCTION
	default:
		return entityv1.BattleReportResolution_BATTLE_REPORT_RESOLUTION_ELIMINATION
	}
}

// EntitiesToBag builds an EntityBag from slices of domain entities.
func EntitiesToBag(users []domain.User, cities []domain.City, buildings []domain.Building, armies []domain.Army) *entityv1.EntityBag {
	bag := &entityv1.EntityBag{}
	for _, u := range users {
		bag.Users = append(bag.Users, UserToProto(u))
	}
	for _, c := range cities {
		bag.Cities = append(bag.Cities, CityToProto(c))
	}
	for _, b := range buildings {
		bag.Buildings = append(bag.Buildings, BuildingToProto(b))
	}
	for _, a := range armies {
		bag.Armies = append(bag.Armies, ArmyToProto(a))
	}
	return bag
}
