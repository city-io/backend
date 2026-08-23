package actors

import (
	"log/slog"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"

	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/messages"
	"cityio/internal/stream"
	"cityio/internal/utils"
)

type armyActor struct {
	baseActor
	Army domain.Army

	ticker       *time.Ticker
	stopTickerCh chan struct{}

	// movesSinceBackup counts tile steps since the last DB enqueue so movement
	// is only persisted every TroopMovementBackupFrequency tiles.
	movesSinceBackup    int
	path                []domain.Coordinates
	movementProgress    time.Duration
	ticksSinceReconcile int
}

func NewArmyActor() BaseActorInterface {
	return &armyActor{}
}

func (state *armyActor) ActorType() string {
	return "army"
}

func (state *armyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case *messages.CreateArmyMessage:
		if state.Army.ArmyID == msg.Army.ArmyID {
			ctx.Respond(messages.Ack{})
			return
		}
		state.Army = msg.Army
		if state.Army.Troops == nil {
			state.Army.Troops = make(map[domain.TroopType]int64)
		}
		if state.Army.DestX != nil && state.Army.DestY != nil && state.Army.MarchID == nil {
			marchID := uuid.NewString()
			state.Army.MarchID = &marchID
		}
		if !msg.Restore {
			if err := state.Store.CreateArmy(state.Ctx(), state.Army); err != nil {
				slog.ErrorContext(state.Ctx(), "failed to persist army create", "army_id", state.Army.ArmyID, "error", err)
				ctx.Respond(&messages.InternalError{})
				ctx.Stop(ctx.Self())
				return
			}
		}
		state.addTile(state.Army.X, state.Army.Y)
		state.recordExploration(state.Army.Owner, domain.Vision{Armies: []domain.Army{state.Army}})
		state.updateUpkeepCity()
		state.restorePath()
		state.startPeriodicOperation(ctx)
		if !msg.Restore {
			state.publish()
		}
		ctx.Respond(messages.Ack{})

	case messages.GetArmyMessage:
		army := state.Army
		army.RemainingPath = append([]domain.Coordinates(nil), state.path...)
		ctx.Respond(&messages.GetArmyResponseMessage{Army: army})

	case messages.MoveArmyMessage:
		x, y := clampCoord(msg.X), clampCoord(msg.Y)
		if state.Army.X == x && state.Army.Y == y {
			state.clearMarch()
			state.Store.EnqueueArmy(state.Army)
			state.publish()
			ctx.Respond(messages.Ack{})
			return
		}
		oldDestX, oldDestY := state.Army.DestX, state.Army.DestY
		oldMarchID := state.Army.MarchID
		oldPath, oldProgress := state.path, state.movementProgress
		marchID := uuid.NewString()
		state.Army.MarchID = &marchID
		state.Army.DestX = &x
		state.Army.DestY = &y
		state.movementProgress = 0
		if err := state.planPath(); err != nil {
			state.Army.DestX, state.Army.DestY = oldDestX, oldDestY
			state.Army.MarchID = oldMarchID
			state.path, state.movementProgress = oldPath, oldProgress
			ctx.Respond(&messages.InternalError{})
			return
		}
		if len(state.path) == 0 {
			state.Army.DestX, state.Army.DestY = oldDestX, oldDestY
			state.Army.MarchID = oldMarchID
			state.path, state.movementProgress = oldPath, oldProgress
			ctx.Respond(&messages.UnreachableDestinationError{X: x, Y: y})
			return
		}
		state.Store.EnqueueArmy(state.Army)
		state.publish()
		ctx.Respond(messages.Ack{})

	case messages.MergeArmiesMessage:
		state.merge(ctx, msg.SourceArmyID)

	case messages.SurrenderTroopsMessage:
		ctx.Respond(&messages.SurrenderTroopsResponseMessage{Troops: state.Army.Troops})
		state.teardown(ctx)

	case messages.DeleteArmyMessage:
		state.teardown(ctx)

	case messages.PeriodicOperationMessage:
		state.step()
	}
}

// merge folds the source army's troops into this one. The source hands them
// over and shuts itself down. Ownership and co-location are validated by the
// caller (the RPC layer) before this runs.
func (state *armyActor) merge(ctx actor.Context, sourceArmyID string) {
	res, err := state.Cluster.Request("army", sourceArmyID, messages.SurrenderTroopsMessage{})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to request troops from source army", "source_army_id", sourceArmyID, "error", err)
		ctx.Respond(err)
		return
	}
	resp, ok := res.(*messages.SurrenderTroopsResponseMessage)
	if !ok {
		ctx.Respond(&messages.InvalidResponseTypeError{})
		return
	}
	for t, c := range resp.Troops {
		state.Army.Troops[t] += c
	}
	state.Store.EnqueueArmy(state.Army)
	state.updateUpkeepCity()
	state.publish()
	ctx.Respond(messages.Ack{})
}

// step advances an army along its terrain-weighted route.
func (state *armyActor) step() {
	state.ticksSinceReconcile++
	if state.Army.DestX == nil || state.Army.DestY == nil {
		state.reconcileTileIfDue()
		return
	}
	destX, destY := *state.Army.DestX, *state.Army.DestY
	if state.Army.X == destX && state.Army.Y == destY {
		state.clearMarch()
		state.Store.EnqueueArmy(state.Army)
		state.publish()
		return
	}
	if len(state.path) == 0 {
		hadMarch := state.Army.MarchID != nil
		if !state.restorePath() {
			if hadMarch && state.Army.MarchID == nil {
				state.publish()
			}
			state.reconcileTileIfDue()
			return
		}
	}
	if !state.advanceMovementClock() {
		state.reconcileTileIfDue()
		return
	}

	oldX, oldY := state.Army.X, state.Army.Y
	next := state.path[0]
	state.path = state.path[1:]
	state.Army.X = next.X
	state.Army.Y = next.Y
	state.removeTile(oldX, oldY)
	state.addTile(state.Army.X, state.Army.Y)
	state.recordExploration(state.Army.Owner, domain.Vision{Armies: []domain.Army{state.Army}})
	state.ticksSinceReconcile = 0
	state.updateUpkeepCity()

	arrived := state.Army.X == destX && state.Army.Y == destY
	marchEnded := arrived
	if arrived {
		state.clearMarch()
	} else if err := state.refreshPath(); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to refresh army route", "army_id", state.Army.ArmyID, "error", err)
		state.path = nil
	} else if len(state.path) == 0 {
		slog.InfoContext(state.Ctx(), "army stopped at the edge of known traversable terrain", "army_id", state.Army.ArmyID, "x", destX, "y", destY)
		state.clearMarch()
		marchEnded = true
	}
	state.movesSinceBackup++
	if marchEnded || state.movesSinceBackup >= constants.TroopMovementBackupFrequency {
		state.Store.EnqueueArmy(state.Army)
		state.movesSinceBackup = 0
	}
	state.publish()
}

func (state *armyActor) restorePath() bool {
	if state.Army.DestX == nil || state.Army.DestY == nil {
		return true
	}
	if err := state.planPath(); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to restore army route", "army_id", state.Army.ArmyID, "error", err)
		return false
	}
	if len(state.path) == 0 && (state.Army.X != *state.Army.DestX || state.Army.Y != *state.Army.DestY) {
		destination := domain.Coordinates{X: *state.Army.DestX, Y: *state.Army.DestY}
		slog.InfoContext(state.Ctx(), "army stopped at the edge of known traversable terrain", "army_id", state.Army.ArmyID, "x", destination.X, "y", destination.Y)
		state.clearMarch()
		state.Store.EnqueueArmy(state.Army)
		return false
	}
	return true
}

func (state *armyActor) clearMarch() {
	state.Army.DestX = nil
	state.Army.DestY = nil
	state.Army.MarchID = nil
	state.path = nil
	state.movementProgress = 0
}

func (state *armyActor) planPath() error {
	if state.Army.DestX == nil || state.Army.DestY == nil {
		state.path = nil
		return nil
	}
	explored, err := state.Store.GetExploredTiles(state.Ctx(), state.Army.Owner)
	if err != nil {
		return err
	}
	known := make(map[domain.Coordinates]struct{}, len(explored))
	for _, coords := range explored {
		known[coords] = struct{}{}
	}
	destination := domain.Coordinates{X: *state.Army.DestX, Y: *state.Army.DestY}
	state.path, _ = domain.FindKnownLandPath(
		state.World.Terrain(), known,
		domain.Coordinates{X: state.Army.X, Y: state.Army.Y}, destination,
	)
	return nil
}

func (state *armyActor) refreshPath() error {
	if state.Army.DestX == nil || state.Army.DestY == nil {
		state.path = nil
		return nil
	}
	explored, err := state.Store.GetExploredTiles(state.Ctx(), state.Army.Owner)
	if err != nil {
		return err
	}
	known := make(map[domain.Coordinates]struct{}, len(explored))
	for _, coords := range explored {
		known[coords] = struct{}{}
	}
	destination := domain.Coordinates{X: *state.Army.DestX, Y: *state.Army.DestY}
	state.path, _ = domain.UpdateKnownLandPath(
		state.World.Terrain(), known,
		domain.Coordinates{X: state.Army.X, Y: state.Army.Y}, destination, state.path,
	)
	return nil
}

func (state *armyActor) currentStepDuration() time.Duration {
	if len(state.path) == 0 {
		return 0
	}
	terrain, ok := state.World.TerrainAt(state.path[0].X, state.path[0].Y)
	if !ok {
		return 0
	}
	return state.baseMovementDuration() * time.Duration(domain.TerrainMovementCost(terrain))
}

func (state *armyActor) advanceMovementClock() bool {
	required := state.currentStepDuration()
	if required <= 0 {
		return false
	}
	state.movementProgress += constants.TroopMovementTickInterval
	if state.movementProgress < required {
		return false
	}
	state.movementProgress -= required
	return true
}

func (state *armyActor) baseMovementDuration() time.Duration {
	duration := time.Duration(0)
	for troopType, count := range state.Army.Troops {
		if count > 0 {
			duration = max(duration, constants.GetTroopMovementDuration(troopType))
		}
	}
	if duration == 0 {
		return constants.GetTroopMovementDuration(domain.TroopTypeSoldier)
	}
	return duration
}

func (state *armyActor) reconcileTileIfDue() {
	if state.ticksSinceReconcile < constants.TroopTileReconcileFrequency {
		return
	}
	state.addTile(state.Army.X, state.Army.Y)
	state.ticksSinceReconcile = 0
}

// upkeepSum is the army's total food upkeep per hour (sum over troop counts).
func (state *armyActor) upkeepSum() int64 {
	var sum int64
	for t, c := range state.Army.Troops {
		sum += c * constants.GetTroopFoodUpkeep(t)
	}
	return sum
}

// updateUpkeepCity attributes this army's food upkeep to the nearest owned city
// (Chebyshev distance), moving the contribution off the previous city when the
// nearest one changes as the army marches.
func (state *armyActor) updateUpkeepCity() {
	cities, err := state.Store.GetCitiesByOwner(state.Ctx(), state.Army.Owner)
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to fetch owned cities for army upkeep", "army_id", state.Army.ArmyID, "error", err)
		return
	}

	var newCityID *string
	best := 0
	for i := range cities {
		d := domain.ChebyshevToCity(cities[i], state.Army.X, state.Army.Y)
		if newCityID == nil || d < best {
			id := cities[i].CityID
			newCityID = &id
			best = d
		}
	}

	old := state.Army.UpkeepCityID
	if old != nil && (newCityID == nil || *old != *newCityID) {
		if err := state.Cluster.Tell("city", *old, messages.RemoveArmyUpkeepMessage{ArmyID: state.Army.ArmyID}); err != nil {
			slog.ErrorContext(state.Ctx(), "failed to remove army upkeep from old city", "army_id", state.Army.ArmyID, "error", err)
		}
	}
	if newCityID != nil {
		if err := state.Cluster.Tell("city", *newCityID, messages.SetArmyUpkeepMessage{ArmyID: state.Army.ArmyID, Amount: state.upkeepSum()}); err != nil {
			slog.ErrorContext(state.Ctx(), "failed to set army upkeep on city", "army_id", state.Army.ArmyID, "error", err)
		}
	}
	state.Army.UpkeepCityID = newCityID
}

func (state *armyActor) addTile(x, y int) {
	if err := state.Cluster.Tell("tile", utils.GetTileIndex(x, y), messages.AddTileArmyMessage{ArmyID: state.Army.ArmyID}); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to add army to tile", "army_id", state.Army.ArmyID, "error", err)
	}
}

func (state *armyActor) removeTile(x, y int) {
	if err := state.Cluster.Tell("tile", utils.GetTileIndex(x, y), messages.RemoveTileArmyMessage{ArmyID: state.Army.ArmyID}); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to remove army from tile", "army_id", state.Army.ArmyID, "error", err)
	}
}

// teardown releases the army's tile presence and upkeep, deletes it from the
// store, notifies the owner's stream, and stops the actor.
func (state *armyActor) teardown(ctx actor.Context) {
	state.stopPeriodicOperation()
	state.removeTile(state.Army.X, state.Army.Y)
	if state.Army.UpkeepCityID != nil {
		if err := state.Cluster.Tell("city", *state.Army.UpkeepCityID, messages.RemoveArmyUpkeepMessage{ArmyID: state.Army.ArmyID}); err != nil {
			slog.ErrorContext(state.Ctx(), "failed to remove army upkeep on teardown", "army_id", state.Army.ArmyID, "error", err)
		}
	}
	if err := state.Store.DeleteArmy(state.Ctx(), state.Army.ArmyID); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to delete army", "army_id", state.Army.ArmyID, "error", err)
	}
	state.publishDeleted()
	slog.DebugContext(state.Ctx(), "shutting down ArmyActor", "army_id", state.Army.ArmyID)
	ctx.Stop(ctx.Self())
}

func (state *armyActor) publish() {
	a := state.Army
	stream.Publish(state.Army.Owner, stream.StateUpdate{Army: &a})
}

func (state *armyActor) publishDeleted() {
	id := state.Army.ArmyID
	stream.Publish(state.Army.Owner, stream.StateUpdate{DeletedArmyID: &id})
}

func (state *armyActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.TroopMovementTickInterval)
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

func (state *armyActor) stopPeriodicOperation() {
	select {
	case <-state.stopTickerCh:
	default:
		close(state.stopTickerCh)
	}
}

// clampCoord bounds a coordinate to the map so a marching order can't target a
// tile off the grid.
func clampCoord(v int) int {
	if v < 0 {
		return 0
	}
	if v > constants.MapSize-1 {
		return constants.MapSize - 1
	}
	return v
}
