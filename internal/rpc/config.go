package rpc

import (
	"context"

	"connectrpc.com/connect"

	"cityio/internal/constants"
	pb "cityio/internal/gen/cityio/v1"
)

type configHandler struct {
	srv *Server
}

func (h *configHandler) GetGameConfig(_ context.Context, _ *connect.Request[pb.GetGameConfigRequest]) (*connect.Response[pb.GetGameConfigResponse], error) {
	return connect.NewResponse(&pb.GetGameConfigResponse{
		MapSize:                     constants.MapSize,
		CitySize:                    constants.CitySize,
		VisionRadius:                constants.VisionRadius,
		BuildingProductionFrequency: constants.BuildingProductionFrequency,
	}), nil
}
