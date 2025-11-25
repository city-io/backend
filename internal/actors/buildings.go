package actors

import (
	"errors"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
)

type buildingActorImpl interface {
	Create(ctx actor.Context, state *buildingActor)
	Destroy(ctx actor.Context, state *buildingActor)
	Handle(ctx actor.Context, state *buildingActor)
}

type buildingActor struct {
	baseActor
	Building models.Building

	Owner *string
	Impl  buildingActorImpl

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func NewBuildingActor() ports.BaseActorInterface {
	return &buildingActor{}
}

func (state *buildingActor) ActorType() string {
	return string(constants.BuildingTypeCityCenter)
}

func (state *buildingActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case messages.CreateBuildingMessage:
		state.Building = msg.Building
		if !msg.Restore {
			ctx.Send(state.Cluster.DB(), messages.CreateBuildingMessage{
				Building: state.Building,
			})
		}
		switch state.Building.BuildingType() {
		case constants.BuildingTypeCityCenter:
			state.Impl = newCityCenterImpl()
		case constants.BuildingTypeTownCenter:
			state.Impl = newTownCenterImpl()
		case constants.BuildingTypeMine:
			state.Impl = newMineImpl()
		case constants.BuildingTypeFarm:
			state.Impl = newFarmImpl()
		case constants.BuildingTypeHouse:
			state.Impl = newHouseImpl()
		}
		// TODO: signal tile actor of building
		state.Impl.Create(ctx, state)
		state.startPeriodicOperation(ctx)
		ctx.Respond(messages.Ack{})

	case messages.UpgradeBuildingMessage:
		if state.Impl != nil {
			state.upgrade(ctx)
		}

	case messages.GetBuildingMessage:
		ctx.Respond(messages.GetBuildingResponseMessage{
			Building: state.Building,
		})

	case messages.DeleteBuildingMessage:
		state.Impl.Destroy(ctx, state)
		state.stopPeriodicOperation()
		state.destroy(ctx)

	default:
		if state.Impl != nil {
			state.Impl.Handle(ctx, state)
		}
	}
}

func (state *buildingActor) constructionActive() bool {
	return (state.Building.Level != state.Building.TargetLevel) || (state.Building.ConstructionStart.Time != nil && state.Building.ConstructionEnd.Time != nil)
}

func (state *buildingActor) upgrade(ctx actor.Context) error {
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

	// TODO: spawn a blocking goroutine that sends a message upon completion
	// ensure that it gets an ACK back for processing
	ctx.Send(state.Cluster.DB(), messages.UpdateBuildingMessage{
		Building: state.Building,
	})
	return nil
}

func (state *buildingActor) destroy(ctx actor.Context) {
	ctx.Send(state.Cluster.DB(), messages.DeleteBuildingMessage{
		BuildingID: state.Building.BuildingID,
	})
	// TODO: notify tile
	state.Log.Debug("shutting down BuildingActor", "building_id", state.Building.BuildingID, "type", state.Building.BuildingType())
	ctx.Stop(ctx.Self())
}

func (state *buildingActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.BuildingProductionFrequency * time.Second)
	state.stopTickerCh = make(chan struct{})

	// TODO: update this to send from root, not ctx
	go func() {
		for {
			select {
			case <-state.ticker.C:
				ctx.Send(ctx.Self(), messages.PeriodicOperationMessage{})
			case <-state.stopTickerCh:
				state.ticker.Stop()
				return
			}
		}
	}()
}

func (state *buildingActor) stopPeriodicOperation() {
	select {
	case <-state.stopTickerCh:
	default:
		close(state.stopTickerCh)
	}
}
