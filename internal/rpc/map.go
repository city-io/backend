package rpc

import (
	"context"

	"connectrpc.com/connect"

	pb "cityio/internal/gen/cityio/v1"
	"cityio/internal/mapping"
)

type mapHandler struct {
	srv *Server
}

func (h *mapHandler) GetMap(ctx context.Context, req *connect.Request[pb.GetMapRequest]) (*connect.Response[pb.GetMapResponse], error) {
	cityList, err := h.srv.store.GetAllCities(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	buildingList, err := h.srv.store.GetAllBuildings(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cities := make([]*pb.City, 0, len(cityList))
	for _, c := range cityList {
		cities = append(cities, mapping.CityToProto(c))
	}
	buildings := make([]*pb.Building, 0, len(buildingList))
	for _, b := range buildingList {
		buildings = append(buildings, mapping.BuildingToProto(b))
	}

	return connect.NewResponse(&pb.GetMapResponse{Cities: cities, Buildings: buildings}), nil
}
