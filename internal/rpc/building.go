package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"cityio/internal/constants"
	"cityio/internal/domain"
	pb "cityio/internal/gen/cityio/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
	"cityio/internal/services"
)

type buildingHandler struct {
	srv *Server
}

func (h *buildingHandler) CreateBuilding(ctx context.Context, req *connect.Request[pb.CreateBuildingRequest]) (*connect.Response[pb.CreateBuildingResponse], error) {
	building, err := services.CreateBuilding(ctx, h.srv.cluster, &services.BuildingInput{
		CityID: req.Msg.GetCityId(),
		Type:   mapping.BuildingTypeFromProto(req.Msg.GetType()),
		X:      int(req.Msg.GetCoords().GetX()),
		Y:      int(req.Msg.GetCoords().GetY()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.CreateBuildingResponse{Building: mapping.BuildingToProto(*building)}), nil
}

func (h *buildingHandler) GetBuilding(ctx context.Context, req *connect.Request[pb.GetBuildingRequest]) (*connect.Response[pb.GetBuildingResponse], error) {
	res, err := h.srv.cluster.Request("building", req.Msg.GetBuildingId(), messages.GetBuildingMessage{})
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

	return connect.NewResponse(&pb.GetBuildingResponse{Building: mapping.BuildingToProto(resp.Building)}), nil
}

func (h *buildingHandler) UpgradeBuilding(ctx context.Context, req *connect.Request[pb.UpgradeBuildingRequest]) (*connect.Response[pb.UpgradeBuildingResponse], error) {
	res, err := h.srv.cluster.Request("building", req.Msg.GetBuildingId(), messages.UpgradeBuildingMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	switch v := res.(type) {
	case messages.Ack:
		return connect.NewResponse(&pb.UpgradeBuildingResponse{}), nil
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

func (h *buildingHandler) DeleteBuilding(ctx context.Context, req *connect.Request[pb.DeleteBuildingRequest]) (*connect.Response[pb.DeleteBuildingResponse], error) {
	if err := h.srv.cluster.Tell("building", req.Msg.GetBuildingId(), messages.DeleteBuildingMessage{BuildingID: req.Msg.GetBuildingId()}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.DeleteBuildingResponse{}), nil
}

func (h *buildingHandler) ListBuildings(ctx context.Context, req *connect.Request[pb.ListBuildingsRequest]) (*connect.Response[pb.ListBuildingsResponse], error) {
	buildingList, err := h.srv.store.GetBuildingsByCity(ctx, req.Msg.GetCityId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	owned, err := h.srv.ownedCities(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	buildingList = domain.FilterBuildings(owned, buildingList, constants.VisionRadius)

	buildings := make([]*pb.Building, 0, len(buildingList))
	for _, b := range buildingList {
		buildings = append(buildings, mapping.BuildingToProto(b))
	}
	return connect.NewResponse(&pb.ListBuildingsResponse{Buildings: buildings}), nil
}
