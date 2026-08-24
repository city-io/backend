package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"

	"cityio/internal/auth"
	"cityio/internal/battles"
	"cityio/internal/constants"
	"cityio/internal/domain"
	entityv1 "cityio/internal/gen/cityio/entity/v1"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
	"cityio/internal/services"
	"cityio/internal/utils"
)

type armyHandler struct {
	srv *Server
}

func (h *armyHandler) getArmy(armyID string) (domain.Army, error) {
	res, err := h.srv.cluster.Request("army", armyID, messages.GetArmyMessage{})
	if err != nil {
		return domain.Army{}, connect.NewError(connect.CodeInternal, err)
	}
	resp, ok := res.(*messages.GetArmyResponseMessage)
	if !ok {
		return domain.Army{}, connect.NewError(connect.CodeNotFound, errors.New("army not found"))
	}
	return resp.Army, nil
}

func (h *armyHandler) requireArmyOwnership(ctx context.Context, armyID string) (domain.Army, error) {
	army, err := h.getArmy(armyID)
	if err != nil {
		return domain.Army{}, err
	}
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return domain.Army{}, connect.NewError(connect.CodeUnauthenticated, errors.New("missing claims"))
	}
	if army.Owner != claims.UserID {
		return domain.Army{}, connect.NewError(connect.CodePermissionDenied, errors.New("army not owned by caller"))
	}
	return army, nil
}

func (h *armyHandler) TrainTroops(ctx context.Context, req *connect.Request[servicev1.TrainTroopsRequest]) (*connect.Response[servicev1.TrainTroopsResponse], error) {
	barracksID := req.Msg.GetBarracksId().GetValue()
	res, err := h.srv.cluster.Request("building", barracksID, messages.GetBuildingMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	building, ok := res.(*messages.GetBuildingResponseMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("barracks not found"))
	}
	if building.Building.BuildingType() != domain.BuildingTypeBarracks {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("building is not a barracks"))
	}
	owns, err := h.srv.ownsCity(ctx, building.Building.CityID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !owns {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("barracks not owned by caller"))
	}

	order, err := services.TrainTroops(ctx, h.srv.cluster, &services.ArmyInput{
		BarracksID: barracksID,
		TroopType:  mapping.TroopTypeFromProto(req.Msg.GetType()),
		Count:      int64(req.Msg.GetCount()),
	})
	if err != nil {
		return nil, trainingError(err)
	}
	return connect.NewResponse(&servicev1.TrainTroopsResponse{Order: mapping.TrainingOrderToProto(*order)}), nil
}

func (h *armyHandler) ListTrainingOrders(ctx context.Context, req *connect.Request[servicev1.ListTrainingOrdersRequest]) (*connect.Response[servicev1.ListTrainingOrdersResponse], error) {
	barracksID := req.Msg.GetBarracksId().GetValue()
	building, err := (&buildingHandler{srv: h.srv}).requireBuildingOwnership(ctx, barracksID)
	if err != nil {
		return nil, err
	}
	if building.BuildingType() != domain.BuildingTypeBarracks {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("building is not a barracks"))
	}
	orders, err := services.GetTrainingOrders(ctx, h.srv.cluster, barracksID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result := make([]*servicev1.TrainingOrder, 0, len(orders))
	for _, order := range orders {
		result = append(result, mapping.TrainingOrderToProto(order))
	}
	return connect.NewResponse(&servicev1.ListTrainingOrdersResponse{Orders: result}), nil
}

// trainingError maps the barracks/city rejection errors to Connect codes.
func trainingError(err error) error {
	var insufficientGold *messages.InsufficientGoldError
	var insufficientPop *messages.InsufficientPopulationError
	var capacity *messages.TrainingCapacityExceededError
	var construction *messages.ConstructionInProgressError
	var invalidCount *messages.InvalidTroopCountError
	var invalidType *messages.InvalidTroopTypeError
	switch {
	case errors.As(err, &insufficientGold):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.As(err, &insufficientPop):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.As(err, &capacity):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.As(err, &construction):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.As(err, &invalidCount):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.As(err, &invalidType):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func (h *armyHandler) GetArmy(ctx context.Context, req *connect.Request[servicev1.GetArmyRequest]) (*connect.Response[servicev1.GetArmyResponse], error) {
	army, err := h.getArmy(req.Msg.GetArmyId().GetValue())
	if err != nil {
		return nil, err
	}
	// The owner can always inspect their own army; everyone else needs vision on
	// its tile.
	claims, _ := auth.ClaimsFromContext(ctx)
	if army.Owner != claims.UserID {
		vision, err := h.srv.ownedVision(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if !vision.PointVisible(army.X, army.Y, constants.VisionRadius) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("army not found"))
		}
	}
	protoArmy := mapping.ArmyToProto(army)
	bag := &entityv1.EntityBag{Armies: []*entityv1.Army{protoArmy}}
	if army.Owner == claims.UserID {
		explored, err := h.srv.store.GetExploredTiles(ctx, army.Owner)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if order := h.srv.projectOwnedArmyOrder(army, exploredSet(explored)); order != nil {
			bag.ArmyOrders = append(bag.ArmyOrders, order)
		}
	} else {
		mapping.HidePrivateArmyFields(protoArmy)
	}
	if army.BattleID != nil {
		if battle, ok := battles.Get(*army.BattleID); ok {
			bag.Battles = append(bag.Battles, mapping.BattleToProto(battle, claims.UserID))
		}
	}
	return connect.NewResponse(&servicev1.GetArmyResponse{ArmyId: mapping.ToArmyId(army.ArmyID), Entities: bag}), nil
}

func (h *armyHandler) AttackArmy(ctx context.Context, req *connect.Request[servicev1.AttackArmyRequest]) (*connect.Response[servicev1.AttackArmyResponse], error) {
	armyID, targetID := req.Msg.GetArmyId().GetValue(), req.Msg.GetTargetArmyId().GetValue()
	if armyID == targetID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("army cannot attack itself"))
	}
	if _, err := h.requireArmyOwnership(ctx, armyID); err != nil {
		return nil, err
	}
	target, err := h.getArmy(targetID)
	if err != nil {
		return nil, err
	}
	claims, _ := auth.ClaimsFromContext(ctx)
	if target.Owner == claims.UserID {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("cannot attack an owned army"))
	}
	vision, err := h.srv.ownedVision(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !vision.PointVisible(target.X, target.Y, constants.VisionRadius) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("target army not found"))
	}
	if err := services.AttackArmy(ctx, h.srv.cluster, armyID, targetID); err != nil {
		return nil, armyOrderError(err)
	}
	return connect.NewResponse(&servicev1.AttackArmyResponse{}), nil
}

func (h *armyHandler) ConquerSettlement(ctx context.Context, req *connect.Request[servicev1.ConquerSettlementRequest]) (*connect.Response[servicev1.ConquerSettlementResponse], error) {
	armyID, cityID := req.Msg.GetArmyId().GetValue(), req.Msg.GetCityId().GetValue()
	if _, err := h.requireArmyOwnership(ctx, armyID); err != nil {
		return nil, err
	}
	res, err := h.srv.cluster.Request("city", cityID, messages.GetCityMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	city, ok := res.(*messages.GetCityResponseMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("settlement not found"))
	}
	claims, _ := auth.ClaimsFromContext(ctx)
	if city.City.Owner != nil && *city.City.Owner == claims.UserID {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("settlement already owned"))
	}
	vision, err := h.srv.ownedVision(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !vision.PointVisible(city.City.StartX+city.City.Size/2, city.City.StartY+city.City.Size/2, constants.VisionRadius) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("settlement not found"))
	}
	if err := services.ConquerSettlement(ctx, h.srv.cluster, armyID, cityID); err != nil {
		return nil, armyOrderError(err)
	}
	return connect.NewResponse(&servicev1.ConquerSettlementResponse{}), nil
}

func (h *armyHandler) RetreatArmy(ctx context.Context, req *connect.Request[servicev1.RetreatArmyRequest]) (*connect.Response[servicev1.RetreatArmyResponse], error) {
	armyID := req.Msg.GetArmyId().GetValue()
	if _, err := h.requireArmyOwnership(ctx, armyID); err != nil {
		return nil, err
	}
	if err := services.RetreatArmy(ctx, h.srv.cluster, armyID); err != nil {
		return nil, armyOrderError(err)
	}
	return connect.NewResponse(&servicev1.RetreatArmyResponse{}), nil
}

func armyOrderError(err error) error {
	var unreachable *messages.UnreachableDestinationError
	var inBattle *messages.ArmyInBattleError
	if errors.As(err, &unreachable) || errors.As(err, &inBattle) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func (h *armyHandler) MoveArmy(ctx context.Context, req *connect.Request[servicev1.MoveArmyRequest]) (*connect.Response[servicev1.MoveArmyResponse], error) {
	if req.Msg.Destination == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("destination is required"))
	}
	armyID := req.Msg.GetArmyId().GetValue()
	if _, err := h.requireArmyOwnership(ctx, armyID); err != nil {
		return nil, err
	}
	if err := services.MoveArmy(ctx, h.srv.cluster, armyID, int(req.Msg.GetDestination().GetX()), int(req.Msg.GetDestination().GetY())); err != nil {
		var unreachable *messages.UnreachableDestinationError
		if errors.As(err, &unreachable) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&servicev1.MoveArmyResponse{}), nil
}

func (h *armyHandler) PreviewArmyRoute(ctx context.Context, req *connect.Request[servicev1.PreviewArmyRouteRequest]) (*connect.Response[servicev1.PreviewArmyRouteResponse], error) {
	if req.Msg.Destination == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("destination is required"))
	}
	army, err := h.requireArmyOwnership(ctx, req.Msg.GetArmyId().GetValue())
	if err != nil {
		return nil, err
	}
	x := max(0, min(constants.MapSize-1, int(req.Msg.GetDestination().GetX())))
	y := max(0, min(constants.MapSize-1, int(req.Msg.GetDestination().GetY())))
	explored, err := h.srv.store.GetExploredTiles(ctx, army.Owner)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	known := exploredSet(explored)
	destination := domain.Coordinates{X: x, Y: y}
	route := h.srv.projectArmyRoute(army, destination, known)
	claims, _ := auth.ClaimsFromContext(ctx)
	if tileResult, tileErr := h.srv.cluster.Request("tile", utils.GetTileIndex(x, y), messages.GetTileMessage{}); tileErr == nil {
		if tile, ok := tileResult.(messages.GetTileResponseMessage); ok && tile.CityID != nil {
			if cityResult, cityErr := h.srv.cluster.Request("city", *tile.CityID, messages.GetCityMessage{}); cityErr == nil {
				if response, cityOK := cityResult.(*messages.GetCityResponseMessage); cityOK {
					city := response.City
					center := domain.Coordinates{X: city.StartX + city.Size/2, Y: city.StartY + city.Size/2}
					if center == destination && (city.Owner == nil || *city.Owner != claims.UserID) {
						route = h.srv.projectArmySiegeRoute(army, center, known)
					}
				}
			}
		}
	}
	return connect.NewResponse(&servicev1.PreviewArmyRouteResponse{
		Route: route.route, EstimatedDuration: durationpb.New(route.duration),
	}), nil
}

func (h *armyHandler) MergeArmies(ctx context.Context, req *connect.Request[servicev1.MergeArmiesRequest]) (*connect.Response[servicev1.MergeArmiesResponse], error) {
	targetID := req.Msg.GetTargetArmyId().GetValue()
	sourceID := req.Msg.GetSourceArmyId().GetValue()
	if targetID == sourceID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cannot merge an army into itself"))
	}
	target, err := h.requireArmyOwnership(ctx, targetID)
	if err != nil {
		return nil, err
	}
	source, err := h.requireArmyOwnership(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if target.X != source.X || target.Y != source.Y {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("armies must be on the same tile to merge"))
	}
	result, err := services.MergeArmies(ctx, h.srv.cluster, targetID, sourceID)
	if err != nil {
		var inBattle *messages.ArmyInBattleError
		if errors.As(err, &inBattle) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	bag := mapping.EntitiesToBag(nil, nil, nil, []domain.Army{result.Army})
	if result.Army.OrderID != nil {
		explored, err := h.srv.store.GetExploredTiles(ctx, result.Army.Owner)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if order := h.srv.projectOwnedArmyOrder(result.Army, exploredSet(explored)); order != nil {
			bag.ArmyOrders = append(bag.ArmyOrders, order)
		}
	}
	return connect.NewResponse(&servicev1.MergeArmiesResponse{
		ArmyId:   mapping.ToArmyId(result.Army.ArmyID),
		Entities: bag,
		Deleted:  &entityv1.EntityIdBag{ArmyIds: []*entityv1.ArmyId{mapping.ToArmyId(result.DeletedArmyID)}},
	}), nil
}

func (h *armyHandler) SplitArmy(ctx context.Context, req *connect.Request[servicev1.SplitArmyRequest]) (*connect.Response[servicev1.SplitArmyResponse], error) {
	armyID := req.Msg.GetArmyId().GetValue()
	if _, err := h.requireArmyOwnership(ctx, armyID); err != nil {
		return nil, err
	}
	troops := make(map[domain.TroopType]int64, len(req.Msg.GetTroops()))
	for _, stack := range req.Msg.GetTroops() {
		troopType := mapping.TroopTypeFromProto(stack.GetType())
		if !constants.IsValidTroopType(troopType) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid troop type"))
		}
		if stack.Count == nil || stack.GetCount() <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("split troop counts must be positive"))
		}
		troops[troopType] += int64(stack.GetCount())
	}
	result, err := services.SplitArmy(ctx, h.srv.cluster, armyID, troops)
	if err != nil {
		var invalid *messages.InvalidArmySplitError
		var insufficient *messages.InsufficientTroopsError
		var inBattle *messages.ArmyInBattleError
		switch {
		case errors.As(err, &invalid):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.As(err, &insufficient), errors.As(err, &inBattle):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	bag := mapping.EntitiesToBag(nil, nil, nil, []domain.Army{result.Source, result.Army})
	if result.Source.OrderID != nil {
		explored, err := h.srv.store.GetExploredTiles(ctx, result.Source.Owner)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if order := h.srv.projectOwnedArmyOrder(result.Source, exploredSet(explored)); order != nil {
			bag.ArmyOrders = append(bag.ArmyOrders, order)
		}
	}
	return connect.NewResponse(&servicev1.SplitArmyResponse{ArmyId: mapping.ToArmyId(result.Army.ArmyID), Entities: bag}), nil
}

func (h *armyHandler) ListArmies(ctx context.Context, req *connect.Request[servicev1.ListArmiesRequest]) (*connect.Response[servicev1.ListArmiesResponse], error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing claims"))
	}
	all, err := h.srv.liveArmies(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	owned := make([]domain.Army, 0, len(all))
	for _, a := range all {
		if a.Owner == claims.UserID {
			owned = append(owned, a)
		}
	}
	armyIDs := make([]*entityv1.ArmyId, 0, len(owned))
	for _, a := range owned {
		armyIDs = append(armyIDs, mapping.ToArmyId(a.ArmyID))
	}
	bag := mapping.EntitiesToBag(nil, nil, nil, owned)
	explored, err := h.srv.store.GetExploredTiles(ctx, claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	known := exploredSet(explored)
	for _, army := range owned {
		if order := h.srv.projectOwnedArmyOrder(army, known); order != nil {
			bag.ArmyOrders = append(bag.ArmyOrders, order)
		}
	}
	for _, battle := range battles.All() {
		if battleVisibleToUser(battle, claims.UserID, map[domain.Coordinates]struct{}{}) {
			bag.Battles = append(bag.Battles, mapping.BattleToProto(battle, claims.UserID))
		}
	}
	return connect.NewResponse(&servicev1.ListArmiesResponse{
		ArmyIds:  armyIDs,
		Entities: bag,
	}), nil
}
