package services

import (
	"context"
	"log/slog"

	"cityio/internal/contracts"
	"cityio/internal/domain"
	"cityio/internal/messages"
)

func RestoreArmy(ctx context.Context, cluster contracts.ClusterProvider, army *domain.Army) error {
	if _, err := cluster.Request("army", army.ArmyID, &messages.CreateArmyMessage{Army: *army, Restore: true}); err != nil {
		slog.ErrorContext(ctx, "failed to restore army actor", "army_id", army.ArmyID, "error", err)
		return err
	}
	return nil
}

// TrainTroops reserves the cost and adds a batch to a city's shared pipeline.
func TrainTroops(ctx context.Context, cluster contracts.ClusterProvider, input *ArmyInput) (*domain.TrainingOrder, error) {
	res, err := cluster.Request("city", input.CityID, messages.TrainTroopsMessage{
		Type:  input.TroopType,
		Count: input.Count,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to request troop training", "city_id", input.CityID, "error", err)
		return nil, err
	}
	switch v := res.(type) {
	case *messages.TrainTroopsResponseMessage:
		return &v.Order, nil
	case error:
		return nil, v
	default:
		return nil, &messages.InvalidResponseTypeError{}
	}
}

// GetTrainingOrders returns queued and active orders for one city.
func GetTrainingOrders(ctx context.Context, cluster contracts.ClusterProvider, cityID string) ([]domain.TrainingOrder, error) {
	res, err := cluster.Request("city", cityID, messages.GetTrainingOrdersMessage{})
	if err != nil {
		slog.ErrorContext(ctx, "failed to request training orders", "city_id", cityID, "error", err)
		return nil, err
	}
	switch response := res.(type) {
	case *messages.GetTrainingOrdersResponseMessage:
		return response.Orders, nil
	case error:
		return nil, response
	default:
		return nil, &messages.InvalidResponseTypeError{}
	}
}

func CancelTrainingOrder(ctx context.Context, cluster contracts.ClusterProvider, cityID, orderID string) error {
	res, err := cluster.Request("city", cityID, messages.CancelTrainingOrderMessage{TrainingOrderID: orderID})
	if err != nil {
		slog.ErrorContext(ctx, "failed to cancel training order", "city_id", cityID, "training_order_id", orderID, "error", err)
		return err
	}
	switch response := res.(type) {
	case messages.Ack:
		return nil
	case error:
		return response
	default:
		return &messages.InvalidResponseTypeError{}
	}
}

// MoveArmy sets an army's marching destination.
func MoveArmy(ctx context.Context, cluster contracts.ClusterProvider, armyID string, x, y int) error {
	res, err := cluster.Request("army", armyID, messages.MoveArmyMessage{X: x, Y: y})
	if err != nil {
		slog.ErrorContext(ctx, "failed to request army move", "army_id", armyID, "error", err)
		return err
	}
	switch v := res.(type) {
	case messages.Ack:
		return nil
	case error:
		return v
	default:
		return &messages.InvalidResponseTypeError{}
	}
}

func AttackArmy(ctx context.Context, cluster contracts.ClusterProvider, armyID, targetArmyID string) error {
	return requestArmyOrder(ctx, cluster, armyID, messages.AttackArmyMessage{TargetArmyID: targetArmyID})
}

func ConquerSettlement(ctx context.Context, cluster contracts.ClusterProvider, armyID, cityID string) error {
	return requestArmyOrder(ctx, cluster, armyID, messages.ConquerSettlementMessage{CityID: cityID})
}

func RetreatArmy(ctx context.Context, cluster contracts.ClusterProvider, armyID string) error {
	return requestArmyOrder(ctx, cluster, armyID, messages.RetreatArmyMessage{})
}

func requestArmyOrder(ctx context.Context, cluster contracts.ClusterProvider, armyID string, message any) error {
	res, err := cluster.Request("army", armyID, message)
	if err != nil {
		slog.ErrorContext(ctx, "failed to request army order", "army_id", armyID, "error", err)
		return err
	}
	switch v := res.(type) {
	case messages.Ack:
		return nil
	case error:
		return v
	default:
		return &messages.InvalidResponseTypeError{}
	}
}

// MergeArmies folds the source army into the target. Both must be validated as
// same-owner and co-located by the caller.
func MergeArmies(ctx context.Context, cluster contracts.ClusterProvider, targetArmyID, sourceArmyID string) (*messages.MergeArmiesResponseMessage, error) {
	res, err := cluster.Request("army", targetArmyID, messages.MergeArmiesMessage{SourceArmyID: sourceArmyID})
	if err != nil {
		slog.ErrorContext(ctx, "failed to request army merge", "target_army_id", targetArmyID, "error", err)
		return nil, err
	}
	switch v := res.(type) {
	case *messages.MergeArmiesResponseMessage:
		return v, nil
	case error:
		return nil, v
	default:
		return nil, &messages.InvalidResponseTypeError{}
	}
}

func SplitArmy(ctx context.Context, cluster contracts.ClusterProvider, armyID string, troops map[domain.TroopType]int64) (*messages.SplitArmyResponseMessage, error) {
	res, err := cluster.Request("army", armyID, messages.SplitArmyMessage{Troops: troops})
	if err != nil {
		slog.ErrorContext(ctx, "failed to request army split", "army_id", armyID, "error", err)
		return nil, err
	}
	switch v := res.(type) {
	case *messages.SplitArmyResponseMessage:
		return v, nil
	case error:
		return nil, v
	default:
		return nil, &messages.InvalidResponseTypeError{}
	}
}
