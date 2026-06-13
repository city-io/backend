package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	pb "cityio/internal/gen/cityio/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
	"cityio/internal/utils"
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

func (h *mapHandler) GetTile(ctx context.Context, req *connect.Request[pb.GetTileRequest]) (*connect.Response[pb.GetTileResponse], error) {
	x := int(req.Msg.GetCoords().GetX())
	y := int(req.Msg.GetCoords().GetY())
	res, err := h.srv.cluster.Request("tile", utils.GetTileIndex(x, y), messages.GetTileMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, ok := res.(messages.GetTileResponseMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tile not found"))
	}
	return connect.NewResponse(&pb.GetTileResponse{
		Tile: mapping.TileToProto(resp.CityID, resp.BuildingID, x, y),
	}), nil
}
