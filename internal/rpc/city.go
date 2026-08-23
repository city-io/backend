package rpc

import (
	"context"
	"errors"
	"math"

	"connectrpc.com/connect"

	"cityio/internal/auth"
	"cityio/internal/constants"
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

	vision, err := h.srv.ownedVision(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !vision.CityVisible(resp.City, constants.VisionRadius) {
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
	city, err := services.CreateCity(ctx, h.srv.cluster, h.srv.store, h.srv.world, &services.CityInput{
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

	cityIDs := make([]*entityv1.CityId, 0, len(cityList))
	for _, c := range cityList {
		cityIDs = append(cityIDs, mapping.ToCityId(c.CityID))
	}

	return connect.NewResponse(&servicev1.ListCitiesResponse{
		CityIds:  cityIDs,
		Entities: mapping.EntitiesToBag(nil, cityList, nil, nil),
	}), nil
}

func (h *cityHandler) UpdateCityPolicy(ctx context.Context, req *connect.Request[servicev1.UpdateCityPolicyRequest]) (*connect.Response[servicev1.UpdateCityPolicyResponse], error) {
	militiaTarget := req.Msg.GetMilitiaTarget()
	taxRatePercent := int(req.Msg.GetTaxRatePercent())
	if err := validateCityPolicy(militiaTarget, taxRatePercent); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	cityID := req.Msg.GetCityId().GetValue()
	owns, err := h.srv.ownsCity(ctx, cityID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !owns {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("city not owned by caller"))
	}
	res, err := h.srv.cluster.Request("city", cityID, messages.UpdateCityPolicyMessage{
		MilitiaTarget:  militiaTarget,
		TaxRatePercent: taxRatePercent,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	switch response := res.(type) {
	case *messages.GetCityResponseMessage:
		return connect.NewResponse(&servicev1.UpdateCityPolicyResponse{City: mapping.CityToProto(response.City)}), nil
	case *messages.InvalidCityPolicyError:
		return nil, connect.NewError(connect.CodeInvalidArgument, response)
	case *messages.CityPolicyLockedError:
		return nil, connect.NewError(connect.CodeFailedPrecondition, response)
	case error:
		return nil, connect.NewError(connect.CodeInternal, response)
	default:
		return nil, connect.NewError(connect.CodeInternal, errors.New("unexpected city policy response"))
	}
}

func validateCityPolicy(militiaTarget float64, taxRatePercent int) error {
	if militiaTarget < 0 || math.Trunc(militiaTarget) != militiaTarget || taxRatePercent < 0 || taxRatePercent > constants.MaxTaxRatePercent {
		return &messages.InvalidCityPolicyError{}
	}
	return nil
}
