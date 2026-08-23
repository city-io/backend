package rpc

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"cityio/internal/constants"
	"cityio/internal/domain"
	entityv1 "cityio/internal/gen/cityio/entity/v1"
	"cityio/internal/mapping"
)

type projectedArmyRoute struct {
	steps    []*entityv1.ArmyRouteStep
	duration time.Duration
}

func (s *Server) projectArmyRoute(army domain.Army, destination domain.Coordinates, explored map[domain.Coordinates]struct{}) projectedArmyRoute {
	path, _ := domain.FindKnownLandPath(
		s.world.Terrain(), explored,
		domain.Coordinates{X: army.X, Y: army.Y}, destination,
	)
	return s.projectArmyPath(army, path, explored)
}

func (s *Server) projectArmyPath(army domain.Army, path []domain.Coordinates, explored map[domain.Coordinates]struct{}) projectedArmyRoute {
	steps := make([]*entityv1.ArmyRouteStep, 0, len(path))
	pathCost := 0
	for _, coords := range path {
		_, known := explored[coords]
		cost := 1
		if known {
			terrain, _ := s.world.TerrainAt(coords.X, coords.Y)
			cost = domain.TerrainMovementCost(terrain)
		}
		pathCost += cost
		steps = append(steps, &entityv1.ArmyRouteStep{
			Coords: &entityv1.Coordinates{X: int32(coords.X), Y: int32(coords.Y)},
		})
	}

	duration := armyMovementDuration(army) * time.Duration(pathCost)
	if duration > 0 {
		duration = ((duration + constants.TroopMovementTickInterval - 1) / constants.TroopMovementTickInterval) * constants.TroopMovementTickInterval
	}
	return projectedArmyRoute{steps: steps, duration: duration}
}

func (s *Server) projectOwnedArmyMarch(army domain.Army, explored map[domain.Coordinates]struct{}) *entityv1.ArmyMarch {
	if army.MarchID == nil || army.DestX == nil || army.DestY == nil {
		return nil
	}
	destination := domain.Coordinates{X: *army.DestX, Y: *army.DestY}
	route := s.projectArmyPath(army, army.RemainingPath, explored)
	if army.RemainingPath == nil {
		route = s.projectArmyRoute(army, destination, explored)
	}
	return &entityv1.ArmyMarch{
		ArmyMarchId:                mapping.ToArmyMarchId(*army.MarchID),
		ArmyId:                     mapping.ToArmyId(army.ArmyID),
		Disclosure:                 entityv1.ArmyMarchDisclosure_ARMY_MARCH_DISCLOSURE_FULL_ROUTE,
		Destination:                &entityv1.Coordinates{X: int32(destination.X), Y: int32(destination.Y)},
		RemainingRoute:             route.steps,
		EstimatedRemainingDuration: durationpb.New(route.duration),
	}
}

func armyMovementDuration(army domain.Army) time.Duration {
	duration := time.Duration(0)
	for troopType, count := range army.Troops {
		if count > 0 {
			duration = max(duration, constants.GetTroopMovementDuration(troopType))
		}
	}
	if duration == 0 {
		return constants.GetTroopMovementDuration(domain.TroopTypeSoldier)
	}
	return duration
}

func exploredSet(coords []domain.Coordinates) map[domain.Coordinates]struct{} {
	explored := make(map[domain.Coordinates]struct{}, len(coords))
	for _, point := range coords {
		explored[point] = struct{}{}
	}
	return explored
}
