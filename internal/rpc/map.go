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
	"cityio/internal/utils"
)

type mapHandler struct {
	srv *Server
}

func (h *mapHandler) GetMap(ctx context.Context, req *connect.Request[pb.GetMapRequest]) (*connect.Response[pb.GetMapResponse], error) {
	owned, err := h.srv.ownedCities(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cityList, err := h.srv.store.GetAllCities(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	buildingList, err := h.srv.store.GetAllBuildings(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cityList = domain.FilterCities(owned, cityList, constants.VisionRadius)
	buildingList = domain.FilterBuildings(owned, buildingList, constants.VisionRadius)

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

	owned, err := h.srv.ownedCities(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !domain.PointVisible(owned, x, y, constants.VisionRadius) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tile not found"))
	}

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
