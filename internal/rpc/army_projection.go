package rpc

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"cityio/internal/constants"
	"cityio/internal/domain"
	entityv1 "cityio/internal/gen/cityio/entity/v1"
	"cityio/internal/mapping"
)

type projectedArmyRoute struct {
	route    *entityv1.ArmyRoute
	duration time.Duration
}

func (s *Server) projectArmyRoute(army domain.Army, destination domain.Coordinates, explored map[domain.Coordinates]struct{}) projectedArmyRoute {
	path, _ := domain.FindKnownLandPath(
		s.world.Terrain(), explored,
		domain.Coordinates{X: army.X, Y: army.Y}, destination,
	)
	return s.projectArmyPath(army, path, explored)
}

func (s *Server) projectArmySiegeRoute(army domain.Army, center domain.Coordinates, explored map[domain.Coordinates]struct{}) projectedArmyRoute {
	path, _ := domain.FindKnownLandPathAdjacent(
		s.world.Terrain(), explored,
		domain.Coordinates{X: army.X, Y: army.Y}, center,
	)
	return s.projectArmyPath(army, path, explored)
}

func (s *Server) projectArmyPath(army domain.Army, path []domain.Coordinates, explored map[domain.Coordinates]struct{}) projectedArmyRoute {
	pathCost := 0
	for _, coords := range path {
		_, known := explored[coords]
		cost := 1
		if known {
			terrain, _ := s.world.TerrainAt(coords.X, coords.Y)
			cost = domain.TerrainMovementCost(terrain)
		}
		pathCost += cost
	}

	known, hiddenEnd := disclosedArmyPath(path, explored)
	route := &entityv1.ArmyRoute{KnownSteps: make([]*entityv1.ArmyRouteStep, 0, len(known))}
	for _, coords := range known {
		route.KnownSteps = append(route.KnownSteps, &entityv1.ArmyRouteStep{
			Coords: &entityv1.Coordinates{X: int32(coords.X), Y: int32(coords.Y)},
		})
	}
	if hiddenEnd != nil {
		route.HiddenSegmentEnd = &entityv1.Coordinates{X: int32(hiddenEnd.X), Y: int32(hiddenEnd.Y)}
	}

	duration := armyMovementDuration(army) * time.Duration(pathCost)
	if duration > 0 {
		duration = ((duration + constants.TroopMovementTickInterval - 1) / constants.TroopMovementTickInterval) * constants.TroopMovementTickInterval
	}
	return projectedArmyRoute{route: route, duration: duration}
}

// disclosedArmyPath separates the contiguous known prefix from the endpoint
// of any hidden remainder so undisclosed geometry is never represented as an
// exact route step.
func disclosedArmyPath(path []domain.Coordinates, explored map[domain.Coordinates]struct{}) ([]domain.Coordinates, *domain.Coordinates) {
	known := make([]domain.Coordinates, 0, len(path))
	for _, coords := range path {
		if _, isExplored := explored[coords]; !isExplored {
			endpoint := path[len(path)-1]
			return known, &endpoint
		}
		known = append(known, coords)
	}
	return known, nil
}

func (s *Server) projectOwnedArmyOrder(army domain.Army, explored map[domain.Coordinates]struct{}) *entityv1.ArmyOrder {
	if army.OrderID == nil {
		return nil
	}
	destination := domain.Coordinates{X: army.X, Y: army.Y}
	if army.DestX != nil && army.DestY != nil {
		destination = domain.Coordinates{X: *army.DestX, Y: *army.DestY}
	}
	route := s.projectArmyPath(army, army.RemainingPath, explored)
	if army.RemainingPath == nil {
		route = s.projectArmyRoute(army, destination, explored)
	}
	order := &entityv1.ArmyOrder{
		ArmyOrderId:                mapping.ToArmyOrderId(*army.OrderID),
		ArmyId:                     mapping.ToArmyId(army.ArmyID),
		RemainingRoute:             route.route,
		EstimatedRemainingDuration: durationpb.New(route.duration),
	}
	coords := &entityv1.Coordinates{X: int32(destination.X), Y: int32(destination.Y)}
	switch army.OrderKind {
	case domain.ArmyOrderAttack:
		if army.TargetArmyID != nil {
			order.Objective = &entityv1.ArmyOrder_AttackArmy{AttackArmy: &entityv1.AttackArmyObjective{TargetArmyId: mapping.ToArmyId(*army.TargetArmyID), LastKnownCoords: coords}}
		}
	case domain.ArmyOrderConquer:
		if army.TargetCityID != nil {
			objective := &entityv1.ConquerSettlementObjective{CityId: mapping.ToCityId(*army.TargetCityID), Destination: coords}
			if army.CaptureStart != nil {
				objective.CaptureStartedAt = timestamppb.New(*army.CaptureStart)
				objective.CaptureDuration = durationpb.New(constants.SettlementCaptureDuration)
			}
			order.Objective = &entityv1.ArmyOrder_ConquerSettlement{ConquerSettlement: objective}
		}
	case domain.ArmyOrderRetreat:
		if army.TargetCityID != nil {
			order.Objective = &entityv1.ArmyOrder_Retreat{Retreat: &entityv1.RetreatObjective{SettlementId: mapping.ToCityId(*army.TargetCityID), Destination: coords}}
		}
	default:
		order.Objective = &entityv1.ArmyOrder_Move{Move: &entityv1.MoveObjective{Destination: coords}}
	}
	return order
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
