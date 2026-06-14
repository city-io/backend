package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"cityio/internal/constants"
	"cityio/internal/domain"
	entityv1 "cityio/internal/gen/cityio/entity/v1"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
	"cityio/internal/services"
)

type buildingHandler struct {
	srv *Server
}

func (h *buildingHandler) requireBuildingOwnership(ctx context.Context, buildingID string) error {
	res, err := h.srv.cluster.Request("building", buildingID, messages.GetBuildingMessage{})
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	resp, ok := res.(messages.GetBuildingResponseMessage)
	if !ok {
		return connect.NewError(connect.CodeNotFound, errors.New("building not found"))
	}
	owns, err := h.srv.ownsCity(ctx, resp.Building.CityID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if !owns {
		return connect.NewError(connect.CodePermissionDenied, errors.New("building not owned by caller"))
	}
	return nil
}

func (h *buildingHandler) CreateBuilding(ctx context.Context, req *connect.Request[servicev1.CreateBuildingRequest]) (*connect.Response[servicev1.CreateBuildingResponse], error) {
	cityID := req.Msg.GetCityId().GetValue()
	owns, err := h.srv.ownsCity(ctx, cityID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !owns {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("city not owned by caller"))
	}
	building, err := services.CreateBuilding(ctx, h.srv.cluster, &services.BuildingInput{
		CityID: cityID,
		Type:   mapping.BuildingTypeFromProto(req.Msg.GetType()),
		X:      int(req.Msg.GetCoords().GetX()),
		Y:      int(req.Msg.GetCoords().GetY()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&servicev1.CreateBuildingResponse{Building: mapping.BuildingToProto(*building)}), nil
}

func (h *buildingHandler) GetBuilding(ctx context.Context, req *connect.Request[servicev1.GetBuildingRequest]) (*connect.Response[servicev1.GetBuildingResponse], error) {
	res, err := h.srv.cluster.Request("building", req.Msg.GetBuildingId().GetValue(), messages.GetBuildingMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, ok := res.(messages.GetBuildingResponseMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("building not found"))
	}

	owned, err := h.srv.ownedCities(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !domain.PointVisible(owned, resp.Building.X, resp.Building.Y, constants.VisionRadius) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("building not found"))
	}

	return connect.NewResponse(&servicev1.GetBuildingResponse{Building: mapping.BuildingToProto(resp.Building)}), nil
}

func (h *buildingHandler) UpgradeBuilding(ctx context.Context, req *connect.Request[servicev1.UpgradeBuildingRequest]) (*connect.Response[servicev1.UpgradeBuildingResponse], error) {
	bid := req.Msg.GetBuildingId().GetValue()
	if err := h.requireBuildingOwnership(ctx, bid); err != nil {
		return nil, err
	}
	res, err := h.srv.cluster.Request("building", bid, messages.UpgradeBuildingMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	switch v := res.(type) {
	case messages.Ack:
		return connect.NewResponse(&servicev1.UpgradeBuildingResponse{}), nil
	case *messages.InsufficientGoldError:
		return nil, connect.NewError(connect.CodeFailedPrecondition, v)
	case *messages.ConstructionInProgressError:
		return nil, connect.NewError(connect.CodeFailedPrecondition, v)
	case *messages.MaxLevelReachedError:
		return nil, connect.NewError(connect.CodeFailedPrecondition, v)
	case error:
		return nil, connect.NewError(connect.CodeInternal, v)
	default:
		return nil, connect.NewError(connect.CodeInternal, errors.New("unexpected upgrade response"))
	}
}

func (h *buildingHandler) DeleteBuilding(ctx context.Context, req *connect.Request[servicev1.DeleteBuildingRequest]) (*connect.Response[servicev1.DeleteBuildingResponse], error) {
	bid := req.Msg.GetBuildingId().GetValue()
	if err := h.requireBuildingOwnership(ctx, bid); err != nil {
		return nil, err
	}
	if err := h.srv.cluster.Tell("building", bid, messages.DeleteBuildingMessage{BuildingID: bid}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&servicev1.DeleteBuildingResponse{}), nil
}

func (h *buildingHandler) ListBuildings(ctx context.Context, req *connect.Request[servicev1.ListBuildingsRequest]) (*connect.Response[servicev1.ListBuildingsResponse], error) {
	buildingList, err := h.srv.store.GetBuildingsByCity(ctx, req.Msg.GetCityId().GetValue())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	owned, err := h.srv.ownedCities(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	buildingList = domain.FilterBuildings(owned, buildingList, constants.VisionRadius)

	buildings := make([]*entityv1.Building, 0, len(buildingList))
	for _, b := range buildingList {
		buildings = append(buildings, mapping.BuildingToProto(b))
	}
	return connect.NewResponse(&servicev1.ListBuildingsResponse{Buildings: buildings}), nil
}
