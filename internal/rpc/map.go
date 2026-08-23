package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"cityio/internal/auth"
	"cityio/internal/constants"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
	"cityio/internal/utils"
)

type mapHandler struct {
	srv *Server
}

func (h *mapHandler) GetMap(ctx context.Context, req *connect.Request[servicev1.GetMapRequest]) (*connect.Response[servicev1.GetMapResponse], error) {
	cityList, err := h.srv.store.GetAllCities(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	buildingList, err := h.srv.store.GetAllBuildings(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	armyList, err := h.srv.liveArmies(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	vision, err := h.srv.ownedVisionWithArmies(ctx, armyList)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cityList = vision.FilterCities(cityList, constants.VisionRadius)
	buildingList = vision.FilterBuildings(buildingList, constants.VisionRadius)
	armyList = vision.FilterArmies(armyList, constants.VisionRadius)

	bag := mapping.EntitiesToBag(nil, cityList, buildingList, armyList)
	tileIDs, tiles := mapping.MapTilesToProto(h.srv.world.Terrain(), cityList, buildingList, armyList)
	bag.Tiles = tiles
	// Strip owner-only fields (production/upkeep rates) from any city the caller
	// doesn't own. Population, cap, and starving stay public.
	claims, _ := auth.ClaimsFromContext(ctx)
	for _, c := range bag.GetCities() {
		if c.GetOwner() == nil || c.GetOwner().GetValue() != claims.UserID {
			mapping.HidePrivateCityFields(c)
		}
	}

	return connect.NewResponse(&servicev1.GetMapResponse{
		TileIds:  tileIDs,
		Entities: bag,
	}), nil
}

func (h *mapHandler) GetTile(ctx context.Context, req *connect.Request[servicev1.GetTileRequest]) (*connect.Response[servicev1.GetTileResponse], error) {
	x := int(req.Msg.GetTileId().GetX())
	y := int(req.Msg.GetTileId().GetY())

	vision, err := h.srv.ownedVision(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !vision.PointVisible(x, y, constants.VisionRadius) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tile not found"))
	}
	terrain, ok := h.srv.world.TerrainAt(x, y)
	if !ok {
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
	return connect.NewResponse(&servicev1.GetTileResponse{
		Tile: mapping.TileToProto(resp.CityID, resp.BuildingID, resp.ArmyIDs, terrain, x, y),
	}), nil
}
