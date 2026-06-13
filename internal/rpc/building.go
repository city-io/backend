package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

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
