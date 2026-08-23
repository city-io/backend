package rpc

import (
	"context"
	"sort"
	"strconv"

	"google.golang.org/protobuf/proto"

	"cityio/internal/battles"
	"cityio/internal/constants"
	"cityio/internal/domain"
	entityv1 "cityio/internal/gen/cityio/entity/v1"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
)

type projectedState struct {
	snapshot          *servicev1.StateSnapshot
	existingCities    map[string]struct{}
	existingBuildings map[string]struct{}
	existingArmies    map[string]struct{}
	existingOrders    map[string]struct{}
	existingBattles   map[string]struct{}
}

func (s *Server) buildProjectedState(ctx context.Context, userID string) (*projectedState, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	users := []domain.User{*user}
	if res, err := s.cluster.Request("user", userID, messages.GetUserMessage{}); err == nil {
		if response, ok := res.(*messages.GetUserResponseMessage); ok {
			users[0] = response.User
		}
	}

	cities, err := s.liveCities(ctx)
	if err != nil {
		return nil, err
	}
	buildings, err := s.liveBuildings(ctx)
	if err != nil {
		return nil, err
	}
	armies, err := s.liveArmies(ctx)
	if err != nil {
		return nil, err
	}

	ownedCities := make([]domain.City, 0)
	ownedArmies := make([]domain.Army, 0)
	for _, city := range cities {
		if city.Owner != nil && *city.Owner == userID {
			ownedCities = append(ownedCities, city)
		}
	}
	for _, army := range armies {
		if army.Owner == userID {
			ownedArmies = append(ownedArmies, army)
		}
	}
	vision := domain.Vision{Cities: ownedCities, Armies: ownedArmies}
	visibleCoords := vision.VisibleCoordinates(constants.MapSize, constants.MapSize, constants.VisionRadius)
	visible := make(map[domain.Coordinates]struct{}, len(visibleCoords))
	for _, coords := range visibleCoords {
		visible[coords] = struct{}{}
	}
	if err := s.store.AddExploredTiles(ctx, userID, visibleCoords); err != nil {
		return nil, err
	}
	exploredCoords, err := s.store.GetExploredTiles(ctx, userID)
	if err != nil {
		return nil, err
	}
	sort.Slice(exploredCoords, func(i, j int) bool {
		return exploredCoords[i].Y < exploredCoords[j].Y ||
			(exploredCoords[i].Y == exploredCoords[j].Y && exploredCoords[i].X < exploredCoords[j].X)
	})

	visibleCities := vision.FilterCities(cities, constants.VisionRadius)
	visibleBuildings := vision.FilterBuildings(buildings, constants.VisionRadius)
	visibleArmies := vision.FilterArmies(armies, constants.VisionRadius)
	bag := mapping.EntitiesToBag(users, visibleCities, visibleBuildings, visibleArmies)
	bag.Tiles = mapping.ExploredTilesToProto(s.world.Terrain(), exploredCoords, visible, visibleCities, visibleBuildings, visibleArmies)
	for _, city := range bag.Cities {
		if city.GetOwner() == nil || city.GetOwner().GetValue() != userID {
			mapping.HidePrivateCityFields(city)
		}
	}
	known := exploredSet(exploredCoords)
	for index, army := range visibleArmies {
		if army.Owner != userID {
			mapping.HidePrivateArmyFields(bag.Armies[index])
			continue
		}
		if order := s.projectOwnedArmyOrder(army, known); order != nil {
			bag.ArmyOrders = append(bag.ArmyOrders, order)
		}
	}
	activeBattles := battles.All()
	for _, battle := range activeBattles {
		if battleVisibleToUser(battle, userID, visible) {
			bag.Battles = append(bag.Battles, mapping.BattleToProto(battle))
		}
	}

	tileVisibility := make([]*servicev1.TileVisibility, 0, len(exploredCoords))
	for _, coords := range exploredCoords {
		state := servicev1.TileVisibilityState_TILE_VISIBILITY_STATE_EXPLORED
		if _, ok := visible[coords]; ok {
			state = servicev1.TileVisibilityState_TILE_VISIBILITY_STATE_VISIBLE
		}
		tileVisibility = append(tileVisibility, &servicev1.TileVisibility{
			TileId: mapping.ToTileId(coords.X, coords.Y), State: state,
		})
	}

	projected := &projectedState{
		snapshot:          &servicev1.StateSnapshot{Entities: bag, TileVisibility: tileVisibility},
		existingCities:    make(map[string]struct{}, len(cities)),
		existingBuildings: make(map[string]struct{}, len(buildings)),
		existingArmies:    make(map[string]struct{}, len(armies)),
		existingOrders:    make(map[string]struct{}),
		existingBattles:   make(map[string]struct{}, len(activeBattles)),
	}
	for _, city := range cities {
		projected.existingCities[city.CityID] = struct{}{}
	}
	for _, building := range buildings {
		projected.existingBuildings[building.BuildingID] = struct{}{}
	}
	for _, army := range armies {
		projected.existingArmies[army.ArmyID] = struct{}{}
		if army.OrderID != nil {
			projected.existingOrders[*army.OrderID] = struct{}{}
		}
	}
	for _, battle := range activeBattles {
		projected.existingBattles[battle.BattleID] = struct{}{}
	}
	return projected, nil
}

func diffProjectedState(previous, current *projectedState) *servicev1.StateDelta {
	delta := &servicev1.StateDelta{
		Upserts: &entityv1.EntityBag{}, Deleted: &entityv1.EntityIdBag{}, Hidden: &entityv1.EntityIdBag{},
	}
	prevBag, currBag := previous.snapshot.GetEntities(), current.snapshot.GetEntities()

	diffUsers(prevBag.GetUsers(), currBag.GetUsers(), &delta.Upserts.Users, &delta.Deleted.UserIds)
	diffCities(prevBag.GetCities(), currBag.GetCities(), current.existingCities, &delta.Upserts.Cities, &delta.Deleted.CityIds, &delta.Hidden.CityIds)
	diffBuildings(prevBag.GetBuildings(), currBag.GetBuildings(), current.existingBuildings, &delta.Upserts.Buildings, &delta.Deleted.BuildingIds, &delta.Hidden.BuildingIds)
	diffArmies(prevBag.GetArmies(), currBag.GetArmies(), current.existingArmies, &delta.Upserts.Armies, &delta.Deleted.ArmyIds, &delta.Hidden.ArmyIds)
	diffArmyOrders(prevBag.GetArmyOrders(), currBag.GetArmyOrders(), current.existingOrders, &delta.Upserts.ArmyOrders, &delta.Deleted.ArmyOrderIds, &delta.Hidden.ArmyOrderIds)
	diffBattles(prevBag.GetBattles(), currBag.GetBattles(), current.existingBattles, &delta.Upserts.Battles, &delta.Deleted.BattleIds, &delta.Hidden.BattleIds)
	diffTiles(prevBag.GetTiles(), currBag.GetTiles(), &delta.Upserts.Tiles)

	previousVisibility := make(map[string]servicev1.TileVisibilityState, len(previous.snapshot.TileVisibility))
	for _, visibility := range previous.snapshot.TileVisibility {
		previousVisibility[tileKey(visibility.GetTileId())] = visibility.GetState()
	}
	for _, visibility := range current.snapshot.TileVisibility {
		if previousVisibility[tileKey(visibility.GetTileId())] != visibility.GetState() {
			delta.TileVisibility = append(delta.TileVisibility, visibility)
		}
	}
	return delta
}

func diffArmyOrders(previous, current []*entityv1.ArmyOrder, existing map[string]struct{}, upserts *[]*entityv1.ArmyOrder, deleted, hidden *[]*entityv1.ArmyOrderId) {
	prev := make(map[string]*entityv1.ArmyOrder, len(previous))
	curr := make(map[string]struct{}, len(current))
	for _, entity := range previous {
		prev[entity.GetArmyOrderId().GetValue()] = entity
	}
	for _, entity := range current {
		id := entity.GetArmyOrderId().GetValue()
		curr[id] = struct{}{}
		if old, ok := prev[id]; !ok || !proto.Equal(old, entity) {
			*upserts = append(*upserts, entity)
		}
	}
	for id := range prev {
		if _, ok := curr[id]; ok {
			continue
		}
		if _, exists := existing[id]; exists {
			*hidden = append(*hidden, mapping.ToArmyOrderId(id))
		} else {
			*deleted = append(*deleted, mapping.ToArmyOrderId(id))
		}
	}
}

func diffBattles(previous, current []*entityv1.Battle, existing map[string]struct{}, upserts *[]*entityv1.Battle, deleted, hidden *[]*entityv1.BattleId) {
	prev := make(map[string]*entityv1.Battle, len(previous))
	curr := make(map[string]struct{}, len(current))
	for _, entity := range previous {
		prev[entity.GetBattleId().GetValue()] = entity
	}
	for _, entity := range current {
		id := entity.GetBattleId().GetValue()
		curr[id] = struct{}{}
		if old, ok := prev[id]; !ok || !proto.Equal(old, entity) {
			*upserts = append(*upserts, entity)
		}
	}
	for id := range prev {
		if _, ok := curr[id]; ok {
			continue
		}
		if _, exists := existing[id]; exists {
			*hidden = append(*hidden, mapping.ToBattleId(id))
		} else {
			*deleted = append(*deleted, mapping.ToBattleId(id))
		}
	}
}

func battleVisibleToUser(battle domain.Battle, userID string, visible map[domain.Coordinates]struct{}) bool {
	for _, id := range append(append([]string{}, battle.Attackers.UserIDs...), battle.Defenders.UserIDs...) {
		if id == userID {
			return true
		}
	}
	_, ok := visible[domain.Coordinates{X: battle.X, Y: battle.Y}]
	return ok
}

func diffUsers(previous, current []*entityv1.User, upserts *[]*entityv1.User, deleted *[]*entityv1.UserId) {
	prev := make(map[string]*entityv1.User, len(previous))
	curr := make(map[string]struct{}, len(current))
	for _, entity := range previous {
		prev[entity.GetUserId().GetValue()] = entity
	}
	for _, entity := range current {
		id := entity.GetUserId().GetValue()
		curr[id] = struct{}{}
		if old, ok := prev[id]; !ok || !proto.Equal(old, entity) {
			*upserts = append(*upserts, entity)
		}
	}
	for id := range prev {
		if _, ok := curr[id]; !ok {
			*deleted = append(*deleted, mapping.ToUserId(id))
		}
	}
}

func diffCities(previous, current []*entityv1.City, existing map[string]struct{}, upserts *[]*entityv1.City, deleted, hidden *[]*entityv1.CityId) {
	prev := make(map[string]*entityv1.City, len(previous))
	curr := make(map[string]struct{}, len(current))
	for _, entity := range previous {
		prev[entity.GetCityId().GetValue()] = entity
	}
	for _, entity := range current {
		id := entity.GetCityId().GetValue()
		curr[id] = struct{}{}
		if old, ok := prev[id]; !ok || !proto.Equal(old, entity) {
			*upserts = append(*upserts, entity)
		}
	}
	for id := range prev {
		if _, ok := curr[id]; !ok {
			if _, exists := existing[id]; exists {
				*hidden = append(*hidden, mapping.ToCityId(id))
			} else {
				*deleted = append(*deleted, mapping.ToCityId(id))
			}
		}
	}
}

func diffBuildings(previous, current []*entityv1.Building, existing map[string]struct{}, upserts *[]*entityv1.Building, deleted, hidden *[]*entityv1.BuildingId) {
	prev := make(map[string]*entityv1.Building, len(previous))
	curr := make(map[string]struct{}, len(current))
	for _, entity := range previous {
		prev[entity.GetBuildingId().GetValue()] = entity
	}
	for _, entity := range current {
		id := entity.GetBuildingId().GetValue()
		curr[id] = struct{}{}
		if old, ok := prev[id]; !ok || !proto.Equal(old, entity) {
			*upserts = append(*upserts, entity)
		}
	}
	for id := range prev {
		if _, ok := curr[id]; !ok {
			if _, exists := existing[id]; exists {
				*hidden = append(*hidden, mapping.ToBuildingId(id))
			} else {
				*deleted = append(*deleted, mapping.ToBuildingId(id))
			}
		}
	}
}

func diffArmies(previous, current []*entityv1.Army, existing map[string]struct{}, upserts *[]*entityv1.Army, deleted, hidden *[]*entityv1.ArmyId) {
	prev := make(map[string]*entityv1.Army, len(previous))
	curr := make(map[string]struct{}, len(current))
	for _, entity := range previous {
		prev[entity.GetArmyId().GetValue()] = entity
	}
	for _, entity := range current {
		id := entity.GetArmyId().GetValue()
		curr[id] = struct{}{}
		if old, ok := prev[id]; !ok || !proto.Equal(old, entity) {
			*upserts = append(*upserts, entity)
		}
	}
	for id := range prev {
		if _, ok := curr[id]; !ok {
			if _, exists := existing[id]; exists {
				*hidden = append(*hidden, mapping.ToArmyId(id))
			} else {
				*deleted = append(*deleted, mapping.ToArmyId(id))
			}
		}
	}
}

func diffTiles(previous, current []*entityv1.Tile, upserts *[]*entityv1.Tile) {
	prev := make(map[string]*entityv1.Tile, len(previous))
	for _, entity := range previous {
		prev[tileKey(entity.GetTileId())] = entity
	}
	for _, entity := range current {
		if old, ok := prev[tileKey(entity.GetTileId())]; !ok || !proto.Equal(old, entity) {
			*upserts = append(*upserts, entity)
		}
	}
}

func tileKey(id *entityv1.TileId) string {
	return strconv.FormatInt(int64(id.GetX()), 10) + ":" + strconv.FormatInt(int64(id.GetY()), 10)
}

func stateDeltaEmpty(delta *servicev1.StateDelta) bool {
	bag, deleted, hidden := delta.GetUpserts(), delta.GetDeleted(), delta.GetHidden()
	return len(bag.GetUsers()) == 0 && len(bag.GetCities()) == 0 && len(bag.GetBuildings()) == 0 && len(bag.GetArmies()) == 0 && len(bag.GetTiles()) == 0 && len(bag.GetArmyOrders()) == 0 && len(bag.GetBattles()) == 0 &&
		len(deleted.GetUserIds()) == 0 && len(deleted.GetCityIds()) == 0 && len(deleted.GetBuildingIds()) == 0 && len(deleted.GetArmyIds()) == 0 && len(deleted.GetTileIds()) == 0 && len(deleted.GetArmyOrderIds()) == 0 && len(deleted.GetBattleIds()) == 0 &&
		len(hidden.GetUserIds()) == 0 && len(hidden.GetCityIds()) == 0 && len(hidden.GetBuildingIds()) == 0 && len(hidden.GetArmyIds()) == 0 && len(hidden.GetTileIds()) == 0 && len(hidden.GetArmyOrderIds()) == 0 && len(hidden.GetBattleIds()) == 0 && len(delta.GetTileVisibility()) == 0
}
