package rpc

import (
	"context"

	"connectrpc.com/connect"

	"cityio/internal/constants"
	servicev1 "cityio/internal/gen/cityio/service/v1"
)

type configHandler struct {
	srv *Server
}

func (h *configHandler) GetGameConfig(_ context.Context, _ *connect.Request[servicev1.GetGameConfigRequest]) (*connect.Response[servicev1.GetGameConfigResponse], error) {
	return connect.NewResponse(&servicev1.GetGameConfigResponse{
		MapSize:                     constants.MapSize,
		CitySize:                    constants.CitySize,
		VisionRadius:                constants.VisionRadius,
		BuildingProductionFrequency: constants.BuildingProductionFrequency,
	}), nil
}
