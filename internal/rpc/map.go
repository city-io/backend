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
	seen, err := h.srv.watchers(ctx)
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

	cityList = domain.FilterCities(seen, cityList, constants.VisionRadius)
	buildingList = domain.FilterBuildings(seen, buildingList, constants.VisionRadius)
	armyList = domain.FilterArmies(seen, armyList, constants.VisionRadius)

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

// GetTerrain bootstraps a client with the map's dimensions and the ground it
// can currently see. Vision is ephemeral — it lasts only while a city, army or
// held structure is watching — so this is a snapshot, not a permanent reveal;
// the state stream sends a fresh set whenever the player's vision moves.
func (h *mapHandler) GetTerrain(ctx context.Context, req *connect.Request[servicev1.GetTerrainRequest]) (*connect.Response[servicev1.GetTerrainResponse], error) {
	w := h.srv.world
	if w == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("world not generated"))
	}

	seen, err := h.srv.watchers(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&servicev1.GetTerrainResponse{
		Width:   int32(w.Width),
		Height:  int32(w.Height),
		Seed:    w.Seed,
		Visible: mapping.VisibleTerrainToProto(w, seen, constants.VisionRadius),
	}), nil
}

func (h *mapHandler) GetTile(ctx context.Context, req *connect.Request[servicev1.GetTileRequest]) (*connect.Response[servicev1.GetTileResponse], error) {
	x := int(req.Msg.GetCoords().GetX())
	y := int(req.Msg.GetCoords().GetY())

	seen, err := h.srv.watchers(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !domain.PointVisible(seen, x, y, constants.VisionRadius) {
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
