package services

import (
	"context"
	"log/slog"

	"cityio/internal/domain"
	"cityio/internal/messages"
	"cityio/internal/ports"
)

func RestoreArmy(ctx context.Context, cluster ports.ClusterProvider, army *domain.Army) error {
	if _, err := cluster.Request("army", army.ArmyID, &messages.CreateArmyMessage{Army: *army, Restore: true}); err != nil {
		slog.ErrorContext(ctx, "failed to restore army actor", "army_id", army.ArmyID, "error", err)
		return err
	}
	return nil
}

// TrainTroops orders a barracks to train a batch of troops. The barracks
// validates capacity, reserves population and deducts gold, then returns the
// durable queue entry.
func TrainTroops(ctx context.Context, cluster ports.ClusterProvider, input *ArmyInput) (*domain.TrainingOrder, error) {
	res, err := cluster.Request("building", input.BarracksID, messages.TrainTroopsMessage{
		Type:  input.TroopType,
		Count: input.Count,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to request troop training", "barracks_id", input.BarracksID, "error", err)
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

// GetTrainingOrders returns the current FIFO queue for a barracks.
func GetTrainingOrders(ctx context.Context, cluster ports.ClusterProvider, barracksID string) ([]domain.TrainingOrder, error) {
	res, err := cluster.Request("building", barracksID, messages.GetTrainingOrdersMessage{})
	if err != nil {
		slog.ErrorContext(ctx, "failed to request training orders", "barracks_id", barracksID, "error", err)
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

// MoveArmy sets an army's marching destination.
func MoveArmy(ctx context.Context, cluster ports.ClusterProvider, armyID string, x, y int) error {
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

// MergeArmies folds the source army into the target. Both must be validated as
// same-owner and co-located by the caller.
func MergeArmies(ctx context.Context, cluster ports.ClusterProvider, targetArmyID, sourceArmyID string) error {
	res, err := cluster.Request("army", targetArmyID, messages.MergeArmiesMessage{SourceArmyID: sourceArmyID})
	if err != nil {
		slog.ErrorContext(ctx, "failed to request army merge", "target_army_id", targetArmyID, "error", err)
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
