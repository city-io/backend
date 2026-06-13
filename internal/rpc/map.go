package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	pb "cityio/internal/gen/cityio/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
)

type mapHandler struct {
	srv *Server
}

func (h *mapHandler) GetMap(ctx context.Context, req *connect.Request[pb.GetMapRequest]) (*connect.Response[pb.GetMapResponse], error) {
	res, err := h.srv.cluster.RequestDBFuture(messages.GetMapMessage{}).Result()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	snapshot, ok := res.(messages.GetMapResponseMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("unexpected map response"))
	}

	cities := make([]*pb.City, 0, len(snapshot.Cities))
	for _, c := range snapshot.Cities {
		cities = append(cities, mapping.CityToProto(c))
	}
	buildings := make([]*pb.Building, 0, len(snapshot.Buildings))
	for _, b := range snapshot.Buildings {
		buildings = append(buildings, mapping.BuildingToProto(b))
	}

	return connect.NewResponse(&pb.GetMapResponse{Cities: cities, Buildings: buildings}), nil
}
