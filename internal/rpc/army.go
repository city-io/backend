package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"cityio/internal/auth"
	"cityio/internal/constants"
	"cityio/internal/domain"
	entityv1 "cityio/internal/gen/cityio/entity/v1"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
	"cityio/internal/services"
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

	err = services.TrainTroops(ctx, h.srv.cluster, &services.ArmyInput{
		BarracksID: barracksID,
		TroopType:  mapping.TroopTypeFromProto(req.Msg.GetType()),
		Count:      int64(req.Msg.GetCount()),
	})
	if err != nil {
		return nil, trainingError(err)
	}
	return connect.NewResponse(&servicev1.TrainTroopsResponse{}), nil
}

// trainingError maps the barracks/city rejection errors to Connect codes.
func trainingError(err error) error {
	var insufficientGold *messages.InsufficientGoldError
	var insufficientPop *messages.InsufficientPopulationError
	var capacity *messages.TrainingCapacityExceededError
	var construction *messages.ConstructionInProgressError
	var invalidCount *messages.InvalidTroopCountError
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
		owned, err := h.srv.ownedCities(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if !domain.PointVisible(owned, army.X, army.Y, constants.VisionRadius) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("army not found"))
		}
	}
	return connect.NewResponse(&servicev1.GetArmyResponse{Army: mapping.ArmyToProto(army)}), nil
}

func (h *armyHandler) MoveArmy(ctx context.Context, req *connect.Request[servicev1.MoveArmyRequest]) (*connect.Response[servicev1.MoveArmyResponse], error) {
	armyID := req.Msg.GetArmyId().GetValue()
	if _, err := h.requireArmyOwnership(ctx, armyID); err != nil {
		return nil, err
	}
	if err := services.MoveArmy(ctx, h.srv.cluster, armyID, int(req.Msg.GetDestination().GetX()), int(req.Msg.GetDestination().GetY())); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&servicev1.MoveArmyResponse{}), nil
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
	if err := services.MergeArmies(ctx, h.srv.cluster, targetID, sourceID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&servicev1.MergeArmiesResponse{}), nil
}

func (h *armyHandler) ListArmies(ctx context.Context, req *connect.Request[servicev1.ListArmiesRequest]) (*connect.Response[servicev1.ListArmiesResponse], error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing claims"))
	}
	all, err := h.srv.store.GetAllArmies(ctx)
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
	return connect.NewResponse(&servicev1.ListArmiesResponse{
		ArmyIds:  armyIDs,
		Entities: mapping.EntitiesToBag(nil, nil, nil, owned),
	}), nil
}
