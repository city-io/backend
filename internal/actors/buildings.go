package actors

import (
	"errors"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"
)

type BuildingActor struct {
	BaseActor
	Building models.Building

	Owner *string
}

func (state *BuildingActor) constructionActive() bool {
	return (state.Building.Level != state.Building.TargetLevel) || (state.Building.ConstructionStart.Time != nil && state.Building.ConstructionEnd.Time != nil)
}

func (state *BuildingActor) upgrade(ctx actor.Context) error {
	if state.Owner == nil {
		return errors.New("cannot upgrade building without owner")
	}
	if state.constructionActive() {
		return &messages.ConstructionInProgressError{BuildingID: state.Building.BuildingID}
	}
	buildingType := state.Building.BuildingType()
	if state.Building.Level >= constants.MAX_BUILDING_LEVEL {
		return &messages.MaxLevelReachedError{BuildingID: state.Building.BuildingID}
	}

	res, err := state.Cluster.Request("user", *state.Owner, messages.CheckAndDeductGoldMessage{
		Amount: constants.GetBuildingCost(buildingType, state.Building.Level),
	})
	if err != nil {
		state.Log.Error("failed to check user balance for upgrade", "error", err)
		return err
	}
	switch msg := res.(type) {
	case messages.Ack:
		// continue upgrade
	case messages.InsufficientGoldError:
		state.Log.Warn("not enough gold", "needed", msg.Missing)
		return &msg
	default:
		state.Log.Error("unexpected response type from user actor", "type", fmt.Sprintf("%T", res))
		return fmt.Errorf("unexpected response type: %T", res)
	}

	now := time.Now()
	end := now.Add(
		time.Duration(constants.GetBuildingConstructionTime(
			buildingType,
			state.Building.Level,
		)) * time.Second,
	)
	state.Building.TargetLevel++
	state.Building.ConstructionStart = models.NullTime{Time: &now}
	state.Building.ConstructionEnd = models.NullTime{Time: &end}
	ctx.Send(state.Cluster.DB(), messages.UpdateBuildingMessage{
		Building: state.Building,
	})
	return nil
}

func (state *BuildingActor) destroy(ctx actor.Context) {
	ctx.Send(state.Cluster.DB(), messages.DeleteBuildingMessage{
		BuildingID: state.Building.BuildingID,
	})
	state.Log.Debug("shutting down BuildingActor", "building_id", state.Building.BuildingID, "type", state.Building.BuildingType())
	ctx.Stop(ctx.Self())
}
