package rpc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"cityio/internal/auth"
	pb "cityio/internal/gen/cityio/v1"
	"cityio/internal/mapping"
	"cityio/internal/messages"
	"cityio/internal/services"
)

type cityHandler struct {
	srv *Server
}

func (h *cityHandler) GetCity(ctx context.Context, req *connect.Request[pb.GetCityRequest]) (*connect.Response[pb.GetCityResponse], error) {
	res, err := h.srv.cluster.Request("city", req.Msg.GetCityId(), messages.GetCityMessage{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, ok := res.(*messages.GetCityResponseMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("city not found"))
	}
	return connect.NewResponse(&pb.GetCityResponse{City: mapping.CityToProto(resp.City)}), nil
}

func (h *cityHandler) CreateCity(ctx context.Context, req *connect.Request[pb.CreateCityRequest]) (*connect.Response[pb.CreateCityResponse], error) {
	city, err := services.CreateCity(ctx, h.srv.cluster, h.srv.store, &services.CityInput{
		Type:  mapping.CityTypeFromProto(req.Msg.GetType()),
		Owner: req.Msg.Owner,
		Name:  req.Msg.GetName(),
		Size:  int(req.Msg.GetSize()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.CreateCityResponse{City: mapping.CityToProto(*city)}), nil
}

func (h *cityHandler) ListCities(ctx context.Context, req *connect.Request[pb.ListCitiesRequest]) (*connect.Response[pb.ListCitiesResponse], error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing claims"))
	}
	cityList, err := h.srv.store.GetCitiesByOwner(ctx, claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	cities := make([]*pb.City, 0, len(cityList))
	for _, c := range cityList {
		cities = append(cities, mapping.CityToProto(c))
	}
	return connect.NewResponse(&pb.ListCitiesResponse{Cities: cities}), nil
}
