package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"cityio/internal/auth"
	"cityio/internal/constants"
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
	claims, _ := auth.ClaimsFromContext(ctx)
	state, err := h.srv.buildProjectedState(ctx, claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	tileIDs := make([]*entityv1.TileId, 0, constants.MapSize*constants.MapSize)
	for y := 0; y < constants.MapSize; y++ {
		for x := 0; x < constants.MapSize; x++ {
			tileIDs = append(tileIDs, mapping.ToTileId(x, y))
		}
	}

	return connect.NewResponse(&servicev1.GetMapResponse{
		TileIds:        tileIDs,
		Entities:       state.snapshot.Entities,
		TileVisibility: state.snapshot.TileVisibility,
	}), nil
}

func (h *mapHandler) GetTile(ctx context.Context, req *connect.Request[servicev1.GetTileRequest]) (*connect.Response[servicev1.GetTileResponse], error) {
	x := int(req.Msg.GetTileId().GetX())
	y := int(req.Msg.GetTileId().GetY())

	claims, _ := auth.ClaimsFromContext(ctx)
	explored, err := h.srv.store.GetExploredTiles(ctx, claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	known := false
	for _, coords := range explored {
		if coords.X == x && coords.Y == y {
			known = true
			break
		}
	}
	if !known {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tile not found"))
	}
	terrain, ok := h.srv.world.TerrainAt(x, y)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("tile not found"))
	}

	vision, err := h.srv.ownedVision(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	visible := vision.PointVisible(x, y, constants.VisionRadius)
	var tile *entityv1.Tile
	if visible {
		res, err := h.srv.cluster.Request("tile", utils.GetTileIndex(x, y), messages.GetTileMessage{})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		resp, ok := res.(messages.GetTileResponseMessage)
		if !ok {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("tile not found"))
		}
		tile = mapping.TileToProto(resp.CityID, resp.BuildingID, resp.ArmyIDs, terrain, x, y)
	} else {
		tile = mapping.TileToProto(nil, nil, nil, terrain, x, y)
	}
	visibility := servicev1.TileVisibilityState_TILE_VISIBILITY_STATE_EXPLORED
	if visible {
		visibility = servicev1.TileVisibilityState_TILE_VISIBILITY_STATE_VISIBLE
	}
	return connect.NewResponse(&servicev1.GetTileResponse{
		Tile:       tile,
		Visibility: &servicev1.TileVisibility{TileId: mapping.ToTileId(x, y), State: visibility},
	}), nil
}
