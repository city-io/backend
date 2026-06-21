package rpc

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/durationpb"

	"cityio/internal/constants"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/mapping"
)

type configHandler struct {
	srv *Server
}

func (h *configHandler) GetGameConfig(_ context.Context, _ *connect.Request[servicev1.GetGameConfigRequest]) (*connect.Response[servicev1.GetGameConfigResponse], error) {
	return connect.NewResponse(&servicev1.GetGameConfigResponse{
		MapSize:      constants.MapSize,
		CitySize:     constants.CitySize,
		VisionRadius: constants.VisionRadius,
		BuildingTick: durationpb.New(constants.BuildingTickInterval * time.Second),
		CityTick:     durationpb.New(constants.CityTickInterval * time.Second),
		Buildings:    buildBuildingConfigs(),
	}), nil
}

func buildBuildingConfigs() []*servicev1.BuildingConfig {
	var configs []*servicev1.BuildingConfig
	for _, bt := range constants.AllBuildingTypes() {
		costs := constants.GetBuildingCosts(bt)
		times := constants.GetBuildingConstructionTimes(bt)
		pops := constants.GetBuildingPopulations(bt)
		prodEntries := constants.GetBuildingProductionEntries(bt)

		levels := make([]*servicev1.BuildingLevelStats, constants.MAX_BUILDING_LEVEL)
		for i := range constants.MAX_BUILDING_LEVEL {
			level := &servicev1.BuildingLevelStats{Level: int32(i + 1)}

			if costs != nil {
				level.Cost = []*servicev1.ResourceAmount{{Resource: "gold", Amount: costs[i]}}
			}
			if times != nil {
				level.ConstructionTime = durationpb.New(time.Duration(times[i]) * time.Second)
			}
			if pops != nil {
				level.Population = pops[i]
			}
			for _, entry := range prodEntries {
				level.Production = append(level.Production, &servicev1.ResourceRate{
					Resource: entry.Resource,
					Rate:     mapping.RatePerDay(entry.Amounts[i]),
				})
			}

			levels[i] = level
		}

		configs = append(configs, &servicev1.BuildingConfig{
			Type:   mapping.BuildingTypeToProto(bt),
			Levels: levels,
		})
	}
	return configs
}
