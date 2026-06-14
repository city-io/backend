package actors

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/messages"
	"cityio/internal/utils"
)

type buildingActorImpl interface {
	Create(ctx actor.Context, state *buildingActor)  // on-create hook for building-specific implementation
	Destroy(ctx actor.Context, state *buildingActor) // on-destroy hook for building-specific implementation
	Handle(ctx actor.Context, state *buildingActor)  // custom message handler for building-specific implementation
}

type buildingActor struct {
	baseActor
	Building domain.Building

	Impl buildingActorImpl

	// pending production not yet acknowledged by the city. Accumulated each tick
	// and only cleared once the city acks, so a dropped tick is retried rather
	// than lost.
	pendingGold int64
	pendingFood int64

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func NewBuildingActor() BaseActorInterface {
	return &buildingActor{}
}

func (state *buildingActor) ActorType() string {
	return string(domain.BuildingTypeCityCenter)
}

func (state *buildingActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *messages.CreateBuildingMessage:
		state.Building = msg.Building
		if !msg.Restore {
			if msg.Construct {
				now := time.Now()
				end := now.Add(
					time.Duration(constants.GetBuildingConstructionTime(
						state.Building.BuildingType(),
						1,
					)) * time.Second,
				)
				state.Building.ConstructionStart = domain.NullTime{Time: &now}
				state.Building.ConstructionEnd = domain.NullTime{Time: &end}
				state.Building.Level = 0
				state.Building.TargetLevel = 1
			}

			// TODO: trigger construction complete message
			if err := state.Store.CreateBuilding(state.Ctx(), state.Building); err != nil {
				slog.ErrorContext(state.Ctx(), "failed to persist building create", "building_id", state.Building.BuildingID, "error", err)
			}
		}
		switch state.Building.BuildingType() {
		case domain.BuildingTypeCityCenter:
			state.Impl = newCityCenterImpl()
		case domain.BuildingTypeTownCenter:
			state.Impl = newTownCenterImpl()
		case domain.BuildingTypeMine:
			state.Impl = newMineImpl()
		case domain.BuildingTypeFarm:
			state.Impl = newFarmImpl()
		case domain.BuildingTypeHouse:
			state.Impl = newHouseImpl()
		case domain.BuildingTypeBarracks:
			state.Impl = newBarracksImpl()
		}

		state.Impl.Create(ctx, state)
		_, err := state.Cluster.Request("tile", utils.GetTileIndex(state.Building.X, state.Building.Y), messages.UpdateTileBuildingMessage{
			BuildingID: &state.Building.BuildingID,
		})
		if err != nil {
			slog.ErrorContext(state.Ctx(), "failed to signal tiles of building existence", "error", err)
		}
		state.startPeriodicOperation(ctx)
		ctx.Respond(messages.Ack{})

	case messages.UpgradeBuildingMessage:
		if err := state.upgrade(ctx); err != nil {
			ctx.Respond(err)
			return
		}
		ctx.Respond(messages.Ack{})

	case messages.GetBuildingMessage:
		ctx.Respond(&messages.GetBuildingResponseMessage{
			Building: state.Building,
		})

	case messages.DeleteBuildingMessage:
		state.Impl.Destroy(ctx, state)
		state.stopPeriodicOperation()
		state.destroy(ctx)

	case messages.ReconcileTilesMessage:
		state.reaffirmTile()

	case messages.PeriodicOperationMessage:
		state.reaffirmTile()
		state.checkConstructionComplete()
		if state.Impl != nil {
			state.Impl.Handle(ctx, state)
		}

	default:
		if state.Impl != nil {
			state.Impl.Handle(ctx, state)
		}
	}
}

// reaffirmTile re-pushes this building's presence to its tile. The building's
// coordinates are authoritative; the tile's building index is derived, so this
// idempotent nudge repairs any drift.
func (state *buildingActor) reaffirmTile() {
	if err := state.Cluster.Tell("tile", utils.GetTileIndex(state.Building.X, state.Building.Y), messages.UpdateTileBuildingMessage{
		BuildingID: &state.Building.BuildingID,
	}); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to reaffirm building tile index", "building_id", state.Building.BuildingID, "error", err)
	}
}

func (state *buildingActor) checkConstructionComplete() {
	if !state.constructionActive() {
		return
	}
	if state.Building.ConstructionEnd.Time == nil || time.Now().Before(*state.Building.ConstructionEnd.Time) {
		return
	}
	state.Building.Level = state.Building.TargetLevel
	state.Building.ConstructionStart = domain.NullTime{}
	state.Building.ConstructionEnd = domain.NullTime{}
	state.Store.EnqueueBuilding(state.Building)
	slog.InfoContext(state.Ctx(), "construction complete",
		"building_id", state.Building.BuildingID,
		"type", state.Building.BuildingType(),
		"level", state.Building.Level,
	)
}

func (state *buildingActor) constructionActive() bool {
	return (state.Building.Level != state.Building.TargetLevel) || (state.Building.ConstructionStart.Time != nil && state.Building.ConstructionEnd.Time != nil)
}

func (state *buildingActor) upgrade(ctx actor.Context) error {
	if state.constructionActive() {
		return &messages.ConstructionInProgressError{BuildingID: state.Building.BuildingID}
	}
	buildingType := state.Building.BuildingType()
	if state.Building.Level >= constants.MAX_BUILDING_LEVEL {
		return &messages.MaxLevelReachedError{BuildingID: state.Building.BuildingID}
	}

	res, err := state.Cluster.Request("city", state.Building.CityID, messages.DeductOwnerGoldMessage{
		Amount: constants.GetBuildingCost(buildingType, state.Building.Level),
	})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to deduct gold for upgrade", "error", err)
		return err
	}
	switch msg := res.(type) {
	case messages.Ack:
		// continue upgrade
	case messages.InsufficientGoldError:
		slog.WarnContext(state.Ctx(), "not enough gold", "needed", msg.Missing)
		return &msg
	default:
		slog.ErrorContext(state.Ctx(), "unexpected response type from user actor", "type", fmt.Sprintf("%T", res))
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
	state.Building.ConstructionStart = domain.NullTime{Time: &now}
	state.Building.ConstructionEnd = domain.NullTime{Time: &end}

	// TODO: spawn a blocking goroutine that sends a message upon completion
	// ensure that it gets an ACK back for processing
	state.Store.EnqueueBuilding(state.Building)
	return nil
}

func (state *buildingActor) destroy(ctx actor.Context) {
	if err := state.Store.DeleteBuilding(state.Ctx(), state.Building.BuildingID); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to delete building", "building_id", state.Building.BuildingID, "error", err)
	}
	if _, err := state.Cluster.Request("tile", utils.GetTileIndex(state.Building.X, state.Building.Y), messages.UpdateTileBuildingMessage{
		BuildingID: nil,
	}); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to clear building from tile on destroy", "building_id", state.Building.BuildingID, "error", err)
	}
	slog.DebugContext(state.Ctx(), "shutting down BuildingActor", "building_id", state.Building.BuildingID, "type", state.Building.BuildingType())
	ctx.Stop(ctx.Self())
}

// creditProduction accumulates produced resources and forwards them to the
// city, which credits its owner. The pending total is only cleared once the
// city acks, so a dropped or failed tick is retried on the next one.
func (state *buildingActor) creditProduction(gold, food int64) {
	state.pendingGold += gold
	state.pendingFood += food
	if state.pendingGold == 0 && state.pendingFood == 0 {
		return
	}

	res, err := state.Cluster.Request("city", state.Building.CityID, messages.CreditProductionMessage{
		Gold: state.pendingGold,
		Food: state.pendingFood,
	})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to credit production to city", "error", err)
		return
	}
	if _, ok := res.(messages.Ack); ok {
		state.pendingGold = 0
		state.pendingFood = 0
	}
}

// populationLevel returns the level to use for population/stat lookups, falling
// back to the target level while the building is still under construction
// (level 0) so stat-table indexing stays valid.
func (state *buildingActor) populationLevel() int {
	if state.Building.Level >= 1 {
		return state.Building.Level
	}
	return state.Building.TargetLevel
}

// reportPopulation tells the city this building's absolute contribution to the
// population cap. It is idempotent (keyed by building) and fire-and-forget to
// avoid deadlocking against a city that is mid-create awaiting this building.
func (state *buildingActor) reportPopulation(population float64) {
	if err := state.Cluster.Tell("city", state.Building.CityID, messages.SetBuildingPopulationMessage{
		BuildingID: state.Building.BuildingID,
		Population: population,
	}); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to report building population to city", "error", err)
	}
}

func (state *buildingActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.BuildingProductionFrequency * time.Second)
	state.stopTickerCh = make(chan struct{})

	pid := ctx.Self()
	system := ctx.ActorSystem()
	go func() {
		for {
			select {
			case <-state.ticker.C:
				system.Root.Send(pid, messages.PeriodicOperationMessage{})
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
