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
	"cityio/internal/utils"
)

type mapHandler struct {
	srv *Server
}

func (h *mapHandler) GetMap(ctx context.Context, req *connect.Request[servicev1.GetMapRequest]) (*connect.Response[servicev1.GetMapResponse], error) {
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
	armyList, err := h.srv.store.GetAllArmies(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cityList = domain.FilterCities(owned, cityList, constants.VisionRadius)
	buildingList = domain.FilterBuildings(owned, buildingList, constants.VisionRadius)
	armyList = domain.FilterArmies(owned, armyList, constants.VisionRadius)

	cityIds := make([]*entityv1.CityId, 0, len(cityList))
	for _, c := range cityList {
		cityIds = append(cityIds, mapping.ToCityId(c.CityID))
	}
	buildingIds := make([]*entityv1.BuildingId, 0, len(buildingList))
	for _, b := range buildingList {
		buildingIds = append(buildingIds, mapping.ToBuildingId(b.BuildingID))
	}

	bag := mapping.EntitiesToBag(nil, cityList, buildingList, armyList)
	// Strip owner-only fields (production/upkeep rates) from any city the caller
	// doesn't own. Population, cap, and starving stay public.
	claims, _ := auth.ClaimsFromContext(ctx)
	for _, c := range bag.GetCities() {
		if c.GetOwner() == nil || c.GetOwner().GetValue() != claims.UserID {
			mapping.HidePrivateCityFields(c)
		}
	}

	return connect.NewResponse(&servicev1.GetMapResponse{
		CityIds:     cityIds,
		BuildingIds: buildingIds,
		Entities:    bag,
	}), nil
}

// GetTerrain returns the whole map in one response. Terrain is generated once
// at boot and never changes, so this is deliberately not filtered by vision or
// paged: the planes total a few kilobytes and the client caches them for the
// session. Fog of war hides entities, which is the information that matters —
// the shape of the coastline is not a secret worth a per-viewport query.
func (h *mapHandler) GetTerrain(ctx context.Context, req *connect.Request[servicev1.GetTerrainRequest]) (*connect.Response[servicev1.GetTerrainResponse], error) {
	w := h.srv.world
	if w == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("world not generated"))
	}

	// The world's plane values are numbered to match the proto enums exactly,
	// so these copy straight out with no remapping.
	return connect.NewResponse(&servicev1.GetTerrainResponse{
		Width:   int32(w.Width),
		Height:  int32(w.Height),
		Seed:    w.Seed,
		Terrain: append([]byte(nil), w.Terrain...),
		Relief:  append([]byte(nil), w.Relief...),
		Feature: append([]byte(nil), w.Feature...),
		Special: append([]byte(nil), w.Special...),
		Rivers:  append([]byte(nil), w.Rivers...),
	}), nil
}

func (h *mapHandler) GetTile(ctx context.Context, req *connect.Request[servicev1.GetTileRequest]) (*connect.Response[servicev1.GetTileResponse], error) {
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
	return connect.NewResponse(&servicev1.GetTileResponse{
		Tile: mapping.TileToProto(resp.CityID, resp.BuildingID, resp.ArmyIDs, x, y),
	}), nil
}
