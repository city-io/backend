package rpc

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"cityio/internal/auth"
	"cityio/internal/constants"
	"cityio/internal/domain"
	entityv1 "cityio/internal/gen/cityio/entity/v1"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
	"cityio/internal/services"
	"cityio/internal/utils"
)

type buildingHandler struct {
	srv *Server
}

func (h *buildingHandler) requireBuildingOwnership(ctx context.Context, buildingID string) (domain.Building, error) {
	res, err := h.srv.cluster.Request("building", buildingID, messages.GetBuildingMessage{})
	if err != nil {
		return domain.Building{}, connect.NewError(connect.CodeInternal, err)
	}
	resp, ok := res.(*messages.GetBuildingResponseMessage)
	if !ok {
		return domain.Building{}, connect.NewError(connect.CodeNotFound, errors.New("building not found"))
	}
	claims, _ := auth.ClaimsFromContext(ctx)
	if resp.Building.Owner != "" {
		if resp.Building.Owner != claims.UserID {
			return domain.Building{}, connect.NewError(connect.CodePermissionDenied, errors.New("building not owned by caller"))
		}
		return resp.Building, nil
	}
	owns, err := h.srv.ownsCity(ctx, resp.Building.CityID)
	if err != nil {
		return domain.Building{}, connect.NewError(connect.CodeInternal, err)
	}
	if !owns {
		return domain.Building{}, connect.NewError(connect.CodePermissionDenied, errors.New("building not owned by caller"))
	}
	return resp.Building, nil
}

func (h *buildingHandler) CreateBuilding(ctx context.Context, req *connect.Request[servicev1.CreateBuildingRequest]) (*connect.Response[servicev1.CreateBuildingResponse], error) {
	cityID := req.Msg.GetCityId().GetValue()
	buildingType := mapping.BuildingTypeFromProto(req.Msg.GetType())
	coords := req.Msg.GetCoords()
	if coords == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("building coordinates are required"))
	}
	owner := ""
	if constants.IsStandaloneStructure(buildingType) {
		if cityID != "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("standalone structures cannot belong to a city"))
		}
		claims, _ := auth.ClaimsFromContext(ctx)
		owner = claims.UserID
		if err := h.validateStandalonePlacement(ctx, owner, int(coords.GetX()), int(coords.GetY())); err != nil {
			return nil, err
		}
	} else {
		switch buildingType {
		case domain.BuildingTypeHouse, domain.BuildingTypeFarm, domain.BuildingTypeMine, domain.BuildingTypeBarracks:
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("building type cannot be constructed"))
		}
		owns, err := h.srv.ownsCity(ctx, cityID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if !owns {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("city not owned by caller"))
		}
	}
	building, err := services.CreateBuilding(ctx, h.srv.cluster, &services.BuildingInput{
		CityID: cityID,
		Owner:  owner,
		Type:   buildingType,
		X:      int(coords.GetX()),
		Y:      int(coords.GetY()),
	})
	if err != nil {
		var insufficientGold *messages.InsufficientGoldError
		if errors.As(err, &insufficientGold) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&servicev1.CreateBuildingResponse{Building: mapping.BuildingToProto(*building)}), nil
}

func (h *buildingHandler) validateStandalonePlacement(ctx context.Context, owner string, x, y int) error {
	terrain, ok := h.srv.world.TerrainAt(x, y)
	if !ok {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("structure coordinates are outside the map"))
	}
	if terrain == domain.TerrainTypeWater {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("structures cannot be built on water"))
	}
	res, err := h.srv.cluster.Request("tile", utils.GetTileIndex(x, y), messages.GetTileMessage{})
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	tile, ok := res.(messages.GetTileResponseMessage)
	if !ok {
		return connect.NewError(connect.CodeInternal, errors.New("unexpected tile response"))
	}
	if tile.CityID != nil || tile.BuildingID != nil {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("structure tile must be neutral and free of structures"))
	}
	hasOwnedArmy, err := h.ownedArmyOccupiesTile(owner, x, y, tile.ArmyIDs)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if !hasOwnedArmy {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("an owned army must occupy the structure tile"))
	}
	cities, err := h.srv.ownedCities(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if withinStructurePlacementRange(cities, x, y) {
		return nil
	}
	return connect.NewError(connect.CodeFailedPrecondition, errors.New("structure is outside placement range of an owned settlement"))
}

func (h *buildingHandler) ownedArmyOccupiesTile(owner string, x, y int, armyIDs []string) (bool, error) {
	var requestErr error
	for _, armyID := range armyIDs {
		res, err := h.srv.cluster.Request("army", armyID, messages.GetArmyMessage{})
		if err != nil {
			requestErr = err
			continue
		}
		response, ok := res.(*messages.GetArmyResponseMessage)
		if !ok {
			return false, fmt.Errorf("unexpected army response for %s: %T", armyID, res)
		}
		army := response.Army
		if army.Owner == owner && army.X == x && army.Y == y {
			return true, nil
		}
	}
	return false, requestErr
}

func withinStructurePlacementRange(cities []domain.City, x, y int) bool {
	for _, city := range cities {
		if domain.ChebyshevToCity(city, x, y) <= constants.StructurePlacementRadius {
			return true
		}
	}
	return false
}

func (h *buildingHandler) GetBuilding(ctx context.Context, req *connect.Request[servicev1.GetBuildingRequest]) (*connect.Response[servicev1.GetBuildingResponse], error) {
	res, err := h.srv.cluster.Request("building", req.Msg.GetBuildingId().GetValue(), messages.GetBuildingMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, ok := res.(*messages.GetBuildingResponseMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("building not found"))
	}

	vision, err := h.srv.ownedVision(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !vision.PointVisible(resp.Building.X, resp.Building.Y, constants.VisionRadius) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("building not found"))
	}

	return connect.NewResponse(&servicev1.GetBuildingResponse{Building: mapping.BuildingToProto(resp.Building)}), nil
}

func (h *buildingHandler) UpgradeBuilding(ctx context.Context, req *connect.Request[servicev1.UpgradeBuildingRequest]) (*connect.Response[servicev1.UpgradeBuildingResponse], error) {
	bid := req.Msg.GetBuildingId().GetValue()
	if _, err := h.requireBuildingOwnership(ctx, bid); err != nil {
		return nil, err
	}
	res, err := h.srv.cluster.Request("building", bid, messages.UpgradeBuildingMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	switch v := res.(type) {
	case messages.Ack:
		return connect.NewResponse(&servicev1.UpgradeBuildingResponse{}), nil
	case *messages.InsufficientGoldError:
		return nil, connect.NewError(connect.CodeFailedPrecondition, v)
	case *messages.ConstructionInProgressError:
		return nil, connect.NewError(connect.CodeFailedPrecondition, v)
	case *messages.TrainingInProgressError:
		return nil, connect.NewError(connect.CodeFailedPrecondition, v)
	case *messages.MaxLevelReachedError:
		return nil, connect.NewError(connect.CodeFailedPrecondition, v)
	case error:
		return nil, connect.NewError(connect.CodeInternal, v)
	default:
		return nil, connect.NewError(connect.CodeInternal, errors.New("unexpected upgrade response"))
	}
}

func (h *buildingHandler) DeleteBuilding(ctx context.Context, req *connect.Request[servicev1.DeleteBuildingRequest]) (*connect.Response[servicev1.DeleteBuildingResponse], error) {
	bid := req.Msg.GetBuildingId().GetValue()
	building, err := h.requireBuildingOwnership(ctx, bid)
	if err != nil {
		return nil, err
	}
	switch building.BuildingType() {
	case domain.BuildingTypeCityCenter, domain.BuildingTypeTownCenter:
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("city center and town center cannot be demolished"))
	}
	res, err := h.srv.cluster.Request("building", bid, messages.DeleteBuildingMessage{BuildingID: bid})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	switch response := res.(type) {
	case messages.Ack:
		return connect.NewResponse(&servicev1.DeleteBuildingResponse{}), nil
	case *messages.TrainingInProgressError:
		return nil, connect.NewError(connect.CodeFailedPrecondition, response)
	case error:
		return nil, connect.NewError(connect.CodeInternal, response)
	default:
		return nil, connect.NewError(connect.CodeInternal, errors.New("unexpected delete response"))
	}
}

func (h *buildingHandler) ListBuildings(ctx context.Context, req *connect.Request[servicev1.ListBuildingsRequest]) (*connect.Response[servicev1.ListBuildingsResponse], error) {
	cityID := req.Msg.GetCityId().GetValue()
	var buildingList []domain.Building
	var err error
	if cityID == "" {
		claims, _ := auth.ClaimsFromContext(ctx)
		buildingList, err = h.srv.store.GetBuildingsByOwner(ctx, claims.UserID)
	} else {
		buildingList, err = h.srv.store.GetBuildingsByCity(ctx, cityID)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	vision, err := h.srv.ownedVision(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	buildingList = vision.FilterBuildings(buildingList, constants.VisionRadius)

	buildings := make([]*entityv1.Building, 0, len(buildingList))
	for _, b := range buildingList {
		buildings = append(buildings, mapping.BuildingToProto(b))
	}
	return connect.NewResponse(&servicev1.ListBuildingsResponse{Buildings: buildings}), nil
}
