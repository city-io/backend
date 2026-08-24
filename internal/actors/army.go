package actors

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"

	"cityio/internal/constants"
	"cityio/internal/contracts"
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

func defaultArmyName(armyID string) string {
	suffix := armyID
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	return "Army " + suffix
}

func normalizeArmyName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", &messages.InvalidArmyNameError{Reason: "name is required"}
	}
	if utf8.RuneCountInString(name) > constants.ArmyNameMaxLength {
		return "", &messages.InvalidArmyNameError{Reason: fmt.Sprintf("name must be %d characters or fewer", constants.ArmyNameMaxLength)}
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", &messages.InvalidArmyNameError{Reason: "control characters are not allowed"}
		}
	}
	return name, nil
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
		if strings.TrimSpace(state.Army.Name) == "" {
			state.Army.Name = defaultArmyName(state.Army.ArmyID)
		}
		if state.Army.Troops == nil {
			state.Army.Troops = make(map[domain.TroopType]int64)
		}
		if state.Army.DestX != nil && state.Army.DestY != nil && state.Army.OrderID == nil {
			orderID := uuid.NewString()
			state.Army.OrderID = &orderID
			state.Army.OrderKind = domain.ArmyOrderMove
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
		if !msg.Restore && !msg.SuppressPublish {
			state.publish()
		}
		ctx.Respond(messages.Ack{})

	case messages.GetArmyMessage:
		army := state.Army
		army.RemainingPath = append([]domain.Coordinates(nil), state.path...)
		ctx.Respond(&messages.GetArmyResponseMessage{Army: army})

	case messages.RenameArmyMessage:
		name, err := normalizeArmyName(msg.Name)
		if err != nil {
			ctx.Respond(err)
			return
		}
		if name == state.Army.Name {
			ctx.Respond(&messages.RenameArmyResponseMessage{Army: state.Army})
			return
		}
		if err := state.Store.RenameArmy(state.Ctx(), state.Army.ArmyID, state.Army.Owner, name); err != nil {
			if errors.Is(err, contracts.ErrArmyNameTaken) {
				ctx.Respond(&messages.ArmyNameTakenError{Name: name})
				return
			}
			slog.ErrorContext(state.Ctx(), "failed to rename army", "army_id", state.Army.ArmyID, "error", err)
			ctx.Respond(&messages.InternalError{})
			return
		}
		state.Army.Name = name
		state.Store.EnqueueArmy(state.Army)
		state.publish()
		ctx.Respond(&messages.RenameArmyResponseMessage{Army: state.Army})

	case messages.MoveArmyMessage:
		if state.Army.BattleID != nil {
			ctx.Respond(&messages.ArmyInBattleError{ArmyID: state.Army.ArmyID})
			return
		}
		x, y := clampCoord(msg.X), clampCoord(msg.Y)
		if state.Army.X == x && state.Army.Y == y {
			state.clearOrder()
			state.Store.EnqueueArmy(state.Army)
			state.publish()
			ctx.Respond(messages.Ack{})
			return
		}
		oldDestX, oldDestY := state.Army.DestX, state.Army.DestY
		oldOrderID, oldKind := state.Army.OrderID, state.Army.OrderKind
		oldTargetArmyID, oldTargetCityID, oldCaptureStart := state.Army.TargetArmyID, state.Army.TargetCityID, state.Army.CaptureStart
		oldPath, oldProgress := state.path, state.movementProgress
		orderID := uuid.NewString()
		state.Army.OrderID = &orderID
		state.Army.OrderKind = domain.ArmyOrderMove
		state.Army.TargetArmyID, state.Army.TargetCityID, state.Army.CaptureStart = nil, nil, nil
		state.Army.DestX = &x
		state.Army.DestY = &y
		state.movementProgress = 0
		if err := state.planPath(); err != nil {
			state.Army.DestX, state.Army.DestY = oldDestX, oldDestY
			state.Army.OrderID, state.Army.OrderKind = oldOrderID, oldKind
			state.Army.TargetArmyID, state.Army.TargetCityID, state.Army.CaptureStart = oldTargetArmyID, oldTargetCityID, oldCaptureStart
			state.path, state.movementProgress = oldPath, oldProgress
			ctx.Respond(&messages.InternalError{})
			return
		}
		if len(state.path) == 0 {
			state.Army.DestX, state.Army.DestY = oldDestX, oldDestY
			state.Army.OrderID, state.Army.OrderKind = oldOrderID, oldKind
			state.Army.TargetArmyID, state.Army.TargetCityID, state.Army.CaptureStart = oldTargetArmyID, oldTargetCityID, oldCaptureStart
			state.path, state.movementProgress = oldPath, oldProgress
			ctx.Respond(&messages.UnreachableDestinationError{X: x, Y: y})
			return
		}
		state.Store.EnqueueArmy(state.Army)
		state.publish()
		ctx.Respond(messages.Ack{})

	case messages.AttackArmyMessage:
		state.attack(ctx, msg.TargetArmyID)

	case messages.ConquerSettlementMessage:
		state.conquer(ctx, msg.CityID)

	case messages.RetreatArmyMessage:
		state.retreat(ctx)

	case messages.EnterBattleMessage:
		battleID := msg.BattleID
		state.Army.BattleID = &battleID
		state.path = nil
		state.movementProgress = 0
		if state.Army.OrderKind == domain.ArmyOrderConquer {
			state.Army.CaptureStart = nil
		}
		state.Store.EnqueueArmy(state.Army)
		state.publish()
		if ctx.Sender() != nil {
			ctx.Respond(messages.Ack{})
		}

	case messages.LeaveBattleMessage:
		if state.Army.BattleID != nil && *state.Army.BattleID == msg.BattleID {
			state.Army.BattleID = nil
			if state.Army.OrderKind == domain.ArmyOrderAttack {
				state.clearOrder()
			}
			state.Store.EnqueueArmy(state.Army)
			state.publish()
		}

	case messages.ApplyCasualtiesMessage:
		state.applyCasualties(ctx, msg.Casualties)

	case messages.MergeArmiesMessage:
		state.merge(ctx, msg.SourceArmyID)

	case messages.SplitArmyMessage:
		state.split(ctx, msg.Troops)

	case messages.SurrenderTroopsMessage:
		if state.Army.BattleID != nil {
			ctx.Respond(&messages.ArmyInBattleError{ArmyID: state.Army.ArmyID})
			return
		}
		troops := make(map[domain.TroopType]int64, len(state.Army.Troops))
		for troopType, count := range state.Army.Troops {
			troops[troopType] = count
		}
		state.cleanup()
		ctx.Respond(&messages.SurrenderTroopsResponseMessage{Troops: troops})
		ctx.Stop(ctx.Self())

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
	if state.Army.BattleID != nil {
		ctx.Respond(&messages.ArmyInBattleError{ArmyID: state.Army.ArmyID})
		return
	}
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
	merged := state.Army
	merged.RemainingPath = append([]domain.Coordinates(nil), state.path...)
	stream.Publish(state.Army.Owner, stream.StateUpdate{Army: &merged, DeletedArmyID: &sourceArmyID})
	ctx.Respond(&messages.MergeArmiesResponseMessage{Army: merged, DeletedArmyID: sourceArmyID})
}

func (state *armyActor) split(ctx actor.Context, requested map[domain.TroopType]int64) {
	if state.Army.BattleID != nil {
		ctx.Respond(&messages.ArmyInBattleError{ArmyID: state.Army.ArmyID})
		return
	}
	remaining, detached, err := splitComposition(state.Army.Troops, requested)
	if err != nil {
		ctx.Respond(err)
		return
	}
	newArmy := domain.Army{
		ArmyID: uuid.NewString(),
		Owner:  state.Army.Owner,
		X:      state.Army.X,
		Y:      state.Army.Y,
		Troops: detached,
	}
	newArmy.Name = defaultArmyName(newArmy.ArmyID)
	res, err := state.Cluster.Request("army", newArmy.ArmyID, &messages.CreateArmyMessage{
		Army:            newArmy,
		SuppressPublish: true,
	})
	if err != nil {
		ctx.Respond(err)
		return
	}
	if responseErr, ok := res.(error); ok {
		ctx.Respond(responseErr)
		return
	}
	if _, ok := res.(messages.Ack); !ok {
		ctx.Respond(&messages.InvalidResponseTypeError{})
		return
	}
	state.Army.Troops = remaining
	state.Store.EnqueueArmy(state.Army)
	state.updateUpkeepCity()
	state.publish()
	source := state.Army
	source.RemainingPath = append([]domain.Coordinates(nil), state.path...)
	ctx.Respond(&messages.SplitArmyResponseMessage{Source: source, Army: newArmy})
}

func splitComposition(current, requested map[domain.TroopType]int64) (map[domain.TroopType]int64, map[domain.TroopType]int64, error) {
	if len(requested) == 0 {
		return nil, nil, &messages.InvalidArmySplitError{Reason: "at least one troop is required"}
	}
	remaining := make(map[domain.TroopType]int64, len(current))
	for troopType, count := range current {
		if count > 0 {
			remaining[troopType] = count
		}
	}
	detached := make(map[domain.TroopType]int64, len(requested))
	var detachedCount, remainingCount int64
	for troopType, count := range requested {
		if !constants.IsValidTroopType(troopType) {
			return nil, nil, &messages.InvalidArmySplitError{Reason: fmt.Sprintf("unknown troop type %q", troopType)}
		}
		if count <= 0 {
			return nil, nil, &messages.InvalidArmySplitError{Reason: "troop counts must be positive"}
		}
		available := remaining[troopType]
		if count > available {
			return nil, nil, &messages.InsufficientTroopsError{Type: troopType, Available: available, Requested: count}
		}
		remaining[troopType] -= count
		detached[troopType] = count
		detachedCount += count
	}
	for _, count := range remaining {
		remainingCount += count
	}
	if detachedCount == 0 {
		return nil, nil, &messages.InvalidArmySplitError{Reason: "at least one troop is required"}
	}
	if remainingCount == 0 {
		return nil, nil, &messages.InvalidArmySplitError{Reason: "source army must retain at least one troop"}
	}
	return remaining, detached, nil
}

func (state *armyActor) attack(ctx actor.Context, targetID string) {
	if state.Army.BattleID != nil {
		ctx.Respond(&messages.ArmyInBattleError{ArmyID: state.Army.ArmyID})
		return
	}
	if targetID == state.Army.ArmyID {
		ctx.Respond(&messages.UnreachableDestinationError{X: state.Army.X, Y: state.Army.Y})
		return
	}
	target, err := state.getArmy(targetID)
	if err != nil {
		ctx.Respond(err)
		return
	}
	if target.Owner == state.Army.Owner {
		ctx.Respond(&messages.UnreachableDestinationError{X: target.X, Y: target.Y})
		return
	}
	if target.BattleID != nil {
		if _, err := state.Cluster.Request("battle", *target.BattleID, messages.JoinBattleMessage{Army: state.Army, OpposesArmyID: target.ArmyID}); err != nil {
			ctx.Respond(err)
			return
		}
		battleID := *target.BattleID
		state.Army.BattleID = &battleID
		state.clearOrder()
		state.publish()
		ctx.Respond(messages.Ack{})
		return
	}
	orderID := uuid.NewString()
	state.Army.OrderID = &orderID
	state.Army.OrderKind = domain.ArmyOrderAttack
	state.Army.TargetArmyID = &targetID
	state.Army.TargetCityID, state.Army.CaptureStart = nil, nil
	state.setDestination(target.X, target.Y)
	if state.Army.X == target.X && state.Army.Y == target.Y {
		state.startBattle(target)
	} else if err := state.planPath(); err != nil || len(state.path) == 0 {
		state.clearOrder()
		ctx.Respond(&messages.UnreachableDestinationError{X: target.X, Y: target.Y})
		return
	}
	state.Store.EnqueueArmy(state.Army)
	state.publish()
	ctx.Respond(messages.Ack{})
}

func (state *armyActor) conquer(ctx actor.Context, cityID string) {
	if state.Army.BattleID != nil {
		ctx.Respond(&messages.ArmyInBattleError{ArmyID: state.Army.ArmyID})
		return
	}
	res, err := state.Cluster.Request("city", cityID, messages.GetCityMessage{})
	if err != nil {
		ctx.Respond(err)
		return
	}
	response, ok := res.(*messages.GetCityResponseMessage)
	if !ok {
		ctx.Respond(&messages.CityNotFoundError{CityId: cityID})
		return
	}
	if response.City.Owner != nil && *response.City.Owner == state.Army.Owner {
		ctx.Respond(messages.Ack{})
		return
	}
	x, y := response.City.StartX+response.City.Size/2, response.City.StartY+response.City.Size/2
	explored, err := state.Store.GetExploredTiles(state.Ctx(), state.Army.Owner)
	if err != nil {
		ctx.Respond(&messages.InternalError{})
		return
	}
	known := make(map[domain.Coordinates]struct{}, len(explored))
	for _, coords := range explored {
		known[coords] = struct{}{}
	}
	path, reaches := domain.FindKnownLandPathAdjacent(
		state.World.Terrain(), known,
		domain.Coordinates{X: state.Army.X, Y: state.Army.Y},
		domain.Coordinates{X: x, Y: y},
	)
	if !reaches {
		ctx.Respond(&messages.UnreachableDestinationError{X: x, Y: y})
		return
	}
	destination := domain.Coordinates{X: state.Army.X, Y: state.Army.Y}
	if len(path) > 0 {
		destination = path[len(path)-1]
	}
	orderID := uuid.NewString()
	state.Army.OrderID = &orderID
	state.Army.OrderKind = domain.ArmyOrderConquer
	state.Army.TargetCityID = &cityID
	state.Army.TargetArmyID, state.Army.CaptureStart = nil, nil
	state.setDestination(destination.X, destination.Y)
	state.path = path
	if state.Army.X == destination.X && state.Army.Y == destination.Y {
		state.finishAtDestination()
	}
	state.Store.EnqueueArmy(state.Army)
	state.publish()
	ctx.Respond(messages.Ack{})
}

func (state *armyActor) retreat(ctx actor.Context) {
	if state.Army.BattleID != nil {
		battleID := *state.Army.BattleID
		if _, err := state.Cluster.Request("battle", battleID, messages.RetreatFromBattleMessage{ArmyID: state.Army.ArmyID}); err != nil {
			ctx.Respond(err)
			return
		}
		state.Army.BattleID = nil
	}
	city, ok := state.nearestOwnedSettlement()
	if !ok {
		state.clearOrder()
		state.publish()
		ctx.Respond(messages.Ack{})
		return
	}
	x, y := city.StartX+city.Size/2, city.StartY+city.Size/2
	orderID := uuid.NewString()
	state.Army.OrderID = &orderID
	state.Army.OrderKind = domain.ArmyOrderRetreat
	state.Army.TargetCityID = &city.CityID
	state.Army.TargetArmyID, state.Army.CaptureStart = nil, nil
	state.setDestination(x, y)
	if state.Army.X == x && state.Army.Y == y {
		state.clearOrder()
	} else if err := state.planPath(); err != nil || len(state.path) == 0 {
		state.clearOrder()
		ctx.Respond(&messages.UnreachableDestinationError{X: x, Y: y})
		return
	}
	state.Store.EnqueueArmy(state.Army)
	state.publish()
	ctx.Respond(messages.Ack{})
}

func (state *armyActor) setDestination(x, y int) {
	x, y = clampCoord(x), clampCoord(y)
	state.Army.DestX, state.Army.DestY = &x, &y
	state.path = nil
	state.movementProgress = 0
}

func (state *armyActor) getArmy(id string) (domain.Army, error) {
	res, err := state.Cluster.Request("army", id, messages.GetArmyMessage{})
	if err != nil {
		return domain.Army{}, err
	}
	response, ok := res.(*messages.GetArmyResponseMessage)
	if !ok || response.Army.ArmyID != id {
		return domain.Army{}, &messages.ArmyNotFoundError{ArmyID: id}
	}
	return response.Army, nil
}

func (state *armyActor) refreshAttackTarget() {
	if state.Army.TargetArmyID == nil {
		return
	}
	target, err := state.getArmy(*state.Army.TargetArmyID)
	if err != nil {
		state.clearOrder()
		state.publish()
		return
	}
	if target.BattleID != nil {
		if _, err := state.Cluster.Request("battle", *target.BattleID, messages.JoinBattleMessage{Army: state.Army, OpposesArmyID: target.ArmyID}); err == nil {
			battleID := *target.BattleID
			state.Army.BattleID = &battleID
			state.path = nil
			state.publish()
		}
		return
	}
	vision := domain.Vision{Armies: []domain.Army{state.Army}}
	if cities, err := state.Store.GetCitiesByOwner(state.Ctx(), state.Army.Owner); err == nil {
		vision.Cities = cities
	}
	if !vision.PointVisible(target.X, target.Y, constants.VisionRadius) {
		return
	}
	if state.Army.DestX == nil || state.Army.DestY == nil || *state.Army.DestX != target.X || *state.Army.DestY != target.Y {
		state.setDestination(target.X, target.Y)
		_ = state.planPath()
		state.publish()
	}
}

func (state *armyActor) finishAtDestination() bool {
	switch state.Army.OrderKind {
	case domain.ArmyOrderAttack:
		if state.Army.TargetArmyID != nil {
			if target, err := state.getArmy(*state.Army.TargetArmyID); err == nil && target.X == state.Army.X && target.Y == state.Army.Y {
				state.startBattle(target)
				return true
			}
		}
		state.clearOrder()
	case domain.ArmyOrderConquer:
		if state.engageSettlementDefender() {
			return true
		}
		if state.Army.CaptureStart == nil {
			now := time.Now()
			state.Army.CaptureStart = &now
			state.path = nil
			state.movementProgress = 0
			state.publish()
		}
		return true
	default:
		state.clearOrder()
	}
	state.Store.EnqueueArmy(state.Army)
	state.publish()
	return true
}

func (state *armyActor) handleCapture() bool {
	if state.Army.CaptureStart == nil {
		return false
	}
	if state.engageSettlementDefender() {
		return true
	}
	if time.Since(*state.Army.CaptureStart) < constants.SettlementCaptureDuration {
		return true
	}
	if state.Army.TargetCityID != nil {
		if _, err := state.Cluster.Request("city", *state.Army.TargetCityID, messages.CaptureCityMessage{Owner: state.Army.Owner}); err != nil {
			slog.ErrorContext(state.Ctx(), "failed to capture settlement", "army_id", state.Army.ArmyID, "city_id", *state.Army.TargetCityID, "error", err)
			return true
		}
	}
	state.clearOrder()
	state.Store.EnqueueArmy(state.Army)
	state.updateUpkeepCity()
	state.publish()
	return true
}

func (state *armyActor) engageSettlementDefender() bool {
	res, err := state.Cluster.Request("tile", utils.GetTileIndex(state.Army.X, state.Army.Y), messages.GetTileMessage{})
	if err != nil {
		return false
	}
	tile, ok := res.(messages.GetTileResponseMessage)
	if !ok {
		return false
	}
	var targetCity *domain.City
	atDestination := state.Army.DestX != nil && state.Army.DestY != nil &&
		state.Army.X == *state.Army.DestX && state.Army.Y == *state.Army.DestY
	if state.Army.OrderKind == domain.ArmyOrderConquer && state.Army.TargetCityID != nil && atDestination {
		if cityResult, cityErr := state.Cluster.Request("city", *state.Army.TargetCityID, messages.GetCityMessage{}); cityErr == nil {
			if response, cityOK := cityResult.(*messages.GetCityResponseMessage); cityOK {
				city := response.City
				targetCity = &city
			}
		}
	}
	for _, id := range tile.ArmyIDs {
		if id == state.Army.ArmyID {
			continue
		}
		other, err := state.getArmy(id)
		if err == nil && other.Owner != state.Army.Owner {
			if targetCity != nil && targetCity.MilitiaPopulation >= 1 {
				state.startMilitiaBattle(*targetCity, &other)
			} else {
				state.startBattle(other)
			}
			return true
		}
	}
	if targetCity != nil && targetCity.MilitiaPopulation >= 1 {
		state.startMilitiaBattle(*targetCity, nil)
		return true
	}
	return false
}

func (state *armyActor) startMilitiaBattle(city domain.City, fieldDefender *domain.Army) {
	proposedID := uuid.NewString()
	res, err := state.Cluster.Request("city", city.CityID, messages.BeginMilitiaBattleMessage{BattleID: proposedID})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to enroll settlement militia", "army_id", state.Army.ArmyID, "city_id", city.CityID, "error", err)
		return
	}
	response, ok := res.(*messages.BeginMilitiaBattleResponseMessage)
	if !ok || response.BattleID == "" || response.Count <= 0 {
		return
	}
	if response.BattleID != proposedID {
		if _, err := state.Cluster.Request("battle", response.BattleID, messages.JoinBattleMessage{
			Army:                 state.Army,
			OpposesMilitiaCityID: city.CityID,
		}); err != nil {
			slog.ErrorContext(state.Ctx(), "failed to join militia battle", "army_id", state.Army.ArmyID, "battle_id", response.BattleID, "error", err)
			return
		}
		battleID := response.BattleID
		state.Army.BattleID = &battleID
		state.path = nil
		state.publish()
		if fieldDefender != nil && fieldDefender.BattleID == nil {
			if _, err := state.Cluster.Request("battle", battleID, messages.JoinBattleMessage{
				Army:          *fieldDefender,
				OpposesArmyID: state.Army.ArmyID,
			}); err == nil {
				_ = state.Cluster.Tell("army", fieldDefender.ArmyID, messages.EnterBattleMessage{BattleID: battleID})
			}
		}
		return
	}
	now := time.Now()
	defenders := domain.BattleSide{MilitiaCityID: &city.CityID, MilitiaCount: response.Count}
	if city.Owner != nil {
		defenders.UserIDs = []string{*city.Owner}
	}
	if fieldDefender != nil {
		defenders.UserIDs = appendUnique(defenders.UserIDs, fieldDefender.Owner)
		defenders.ArmyIDs = []string{fieldDefender.ArmyID}
	}
	battle := domain.Battle{
		BattleID:  proposedID,
		X:         state.Army.X,
		Y:         state.Army.Y,
		StartedAt: now,
		NextTick:  now.Add(constants.BattleTickInterval),
		Attackers: domain.BattleSide{UserIDs: []string{state.Army.Owner}, ArmyIDs: []string{state.Army.ArmyID}},
		Defenders: defenders,
	}
	participants := []domain.Army{state.Army}
	if fieldDefender != nil {
		participants = append(participants, *fieldDefender)
	}
	if _, err := state.Cluster.Request("battle", proposedID, &messages.CreateBattleMessage{Battle: battle, Armies: participants}); err != nil {
		_ = state.Cluster.Tell("city", city.CityID, messages.EndMilitiaBattleMessage{BattleID: proposedID})
		slog.ErrorContext(state.Ctx(), "failed to create militia battle", "army_id", state.Army.ArmyID, "city_id", city.CityID, "error", err)
	}
}

func (state *armyActor) startBattle(target domain.Army) {
	if target.BattleID != nil {
		if _, err := state.Cluster.Request("battle", *target.BattleID, messages.JoinBattleMessage{Army: state.Army, OpposesArmyID: target.ArmyID}); err == nil {
			battleID := *target.BattleID
			state.Army.BattleID = &battleID
			state.path = nil
			state.publish()
		}
		return
	}
	battleID := uuid.NewString()
	now := time.Now()
	participants := []domain.Army{state.Army, target}
	battle := domain.Battle{BattleID: battleID, X: state.Army.X, Y: state.Army.Y, StartedAt: now, NextTick: now.Add(constants.BattleTickInterval),
		Attackers: domain.BattleSide{UserIDs: []string{state.Army.Owner}, ArmyIDs: []string{state.Army.ArmyID}},
		Defenders: domain.BattleSide{UserIDs: []string{target.Owner}, ArmyIDs: []string{target.ArmyID}}}
	if res, err := state.Cluster.Request("tile", utils.GetTileIndex(state.Army.X, state.Army.Y), messages.GetTileMessage{}); err == nil {
		if tile, ok := res.(messages.GetTileResponseMessage); ok {
			for _, id := range tile.ArmyIDs {
				if id == state.Army.ArmyID || id == target.ArmyID {
					continue
				}
				army, err := state.getArmy(id)
				if err != nil || army.BattleID != nil {
					continue
				}
				if army.Owner == state.Army.Owner {
					battle.Attackers.ArmyIDs = append(battle.Attackers.ArmyIDs, id)
					participants = append(participants, army)
				}
				if army.Owner == target.Owner {
					battle.Defenders.ArmyIDs = append(battle.Defenders.ArmyIDs, id)
					participants = append(participants, army)
				}
			}
		}
	}
	if _, err := state.Cluster.Request("battle", battleID, &messages.CreateBattleMessage{Battle: battle, Armies: participants}); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to create battle", "army_id", state.Army.ArmyID, "target_army_id", target.ArmyID, "error", err)
	}
}

func (state *armyActor) applyCasualties(ctx actor.Context, casualties map[domain.TroopType]int64) {
	for troopType, count := range casualties {
		removed := min(count, state.Army.Troops[troopType])
		state.Army.Troops[troopType] -= removed
	}
	survived := false
	for _, count := range state.Army.Troops {
		if count > 0 {
			survived = true
			break
		}
	}
	ctx.Respond(&messages.ApplyCasualtiesResponseMessage{ArmyID: state.Army.ArmyID, Survived: survived})
	if !survived {
		state.teardown(ctx)
		return
	}
	state.Store.EnqueueArmy(state.Army)
	state.updateUpkeepCity()
	state.publish()
}

func (state *armyActor) nearestOwnedSettlement() (domain.City, bool) {
	cities, err := state.Store.GetCitiesByOwner(state.Ctx(), state.Army.Owner)
	if err != nil || len(cities) == 0 {
		return domain.City{}, false
	}
	best, distance := cities[0], domain.ChebyshevToCity(cities[0], state.Army.X, state.Army.Y)
	for _, city := range cities[1:] {
		if d := domain.ChebyshevToCity(city, state.Army.X, state.Army.Y); d < distance {
			best, distance = city, d
		}
	}
	return best, true
}

// step advances an army along its terrain-weighted route.
func (state *armyActor) step() {
	state.ticksSinceReconcile++
	if state.Army.BattleID != nil {
		state.reconcileTileIfDue()
		return
	}
	if state.Army.OrderKind == domain.ArmyOrderAttack {
		state.refreshAttackTarget()
	}
	if state.Army.OrderKind == domain.ArmyOrderConquer && state.handleCapture() {
		return
	}
	if state.Army.DestX == nil || state.Army.DestY == nil {
		state.reconcileTileIfDue()
		return
	}
	destX, destY := *state.Army.DestX, *state.Army.DestY
	if state.Army.X == destX && state.Army.Y == destY {
		if state.finishAtDestination() {
			return
		}
		state.clearOrder()
		state.Store.EnqueueArmy(state.Army)
		state.publish()
		return
	}
	if len(state.path) == 0 {
		hadOrder := state.Army.OrderID != nil
		if !state.restorePath() {
			if hadOrder && state.Army.OrderID == nil {
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
	if state.engageSettlementDefender() {
		state.Store.EnqueueArmy(state.Army)
		state.publish()
		return
	}

	arrived := state.Army.X == destX && state.Army.Y == destY
	orderEnded := arrived
	if arrived {
		if state.finishAtDestination() {
			state.publish()
			return
		}
		state.clearOrder()
	} else if err := state.refreshPath(); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to refresh army route", "army_id", state.Army.ArmyID, "error", err)
		state.path = nil
	} else if len(state.path) == 0 {
		slog.InfoContext(state.Ctx(), "army stopped at the edge of known traversable terrain", "army_id", state.Army.ArmyID, "x", destX, "y", destY)
		state.clearOrder()
		orderEnded = true
	}
	state.movesSinceBackup++
	if orderEnded || state.movesSinceBackup >= constants.TroopMovementBackupFrequency {
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
		state.clearOrder()
		state.Store.EnqueueArmy(state.Army)
		return false
	}
	return true
}

func (state *armyActor) clearOrder() {
	state.Army.DestX = nil
	state.Army.DestY = nil
	state.Army.OrderID = nil
	state.Army.OrderKind = ""
	state.Army.TargetArmyID = nil
	state.Army.TargetCityID = nil
	state.Army.CaptureStart = nil
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

// cleanup releases an army's world presence and deletes its persisted state.
// Compound operations call it without publishing so their final mutation can
// produce one coherent stream delta.
func (state *armyActor) cleanup() {
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
}

// teardown cleans up an army, publishes its deletion, and stops the actor.
func (state *armyActor) teardown(ctx actor.Context) {
	state.cleanup()
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
