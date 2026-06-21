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
	"cityio/internal/services"
)

type cityHandler struct {
	srv *Server
}

func (h *cityHandler) GetCity(ctx context.Context, req *connect.Request[servicev1.GetCityRequest]) (*connect.Response[servicev1.GetCityResponse], error) {
	res, err := h.srv.cluster.Request("city", req.Msg.GetCityId().GetValue(), messages.GetCityMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, ok := res.(*messages.GetCityResponseMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("city not found"))
	}

	owned, err := h.srv.ownedCities(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !domain.CityVisible(owned, resp.City, constants.VisionRadius) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("city not found"))
	}

	// Owner gets the full economy intel; everyone else sees only public fields.
	claims, _ := auth.ClaimsFromContext(ctx)
	city := mapping.CityToProto(resp.City)
	if resp.City.Owner == nil || *resp.City.Owner != claims.UserID {
		mapping.HidePrivateCityFields(city)
	}
	return connect.NewResponse(&servicev1.GetCityResponse{City: city}), nil
}

func (h *cityHandler) CreateCity(ctx context.Context, req *connect.Request[servicev1.CreateCityRequest]) (*connect.Response[servicev1.CreateCityResponse], error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing claims"))
	}
	city, err := services.CreateCity(ctx, h.srv.cluster, h.srv.store, &services.CityInput{
		Type:  mapping.CityTypeFromProto(req.Msg.GetType()),
		Owner: &claims.UserID,
		Name:  req.Msg.GetName(),
		Size:  int(req.Msg.GetSize()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&servicev1.CreateCityResponse{City: mapping.CityToProto(*city)}), nil
}

func (h *cityHandler) ListCities(ctx context.Context, req *connect.Request[servicev1.ListCitiesRequest]) (*connect.Response[servicev1.ListCitiesResponse], error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing claims"))
	}
	cityList, err := h.srv.store.GetCitiesByOwner(ctx, claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	cityIds := make([]*entityv1.CityId, 0, len(cityList))
	for _, c := range cityList {
		cityIds = append(cityIds, mapping.ToCityId(c.CityID))
	}

	return connect.NewResponse(&servicev1.ListCitiesResponse{
		CityIds:  cityIds,
		Entities: mapping.EntitiesToBag(nil, cityList, nil),
	}), nil
}
