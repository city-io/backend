package actors

import (
	"log/slog"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"

	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/messages"
)

// trainingBatch is one queued training order. completeAt is zero until the
// batch reaches the front of the queue and starts training.
type trainingBatch struct {
	troopType  domain.TroopType
	count      int64
	completeAt time.Time
}

// barracksImpl trains troops. Each barracks trains one batch at a time (up to
// its level capacity); further orders queue FIFO. A one-shot timer fires a
// PeriodicOperationMessage when the front batch finishes, mirroring the
// construction-completion pattern; the per-tick poll is the idempotent safety
// net.
type barracksImpl struct {
	queue []trainingBatch
	timer *time.Timer
}

func newBarracksImpl() buildingActorImpl {
	return &barracksImpl{}
}

func (b *barracksImpl) Create(ctx actor.Context, state *buildingActor) {}

func (b *barracksImpl) Destroy(ctx actor.Context, state *buildingActor) {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
}

func (b *barracksImpl) Handle(ctx actor.Context, state *buildingActor) {
	switch msg := ctx.Message().(type) {
	case messages.TrainTroopsMessage:
		b.handleTrain(ctx, state, msg)
	case messages.PeriodicOperationMessage:
		b.checkComplete(ctx, state)
	}
}

func (b *barracksImpl) handleTrain(ctx actor.Context, state *buildingActor, msg messages.TrainTroopsMessage) {
	if state.constructionActive() {
		ctx.Respond(&messages.ConstructionInProgressError{BuildingID: state.Building.BuildingID})
		return
	}
	if msg.Count <= 0 || !constants.IsValidTroopType(msg.Type) {
		ctx.Respond(&messages.InvalidTroopCountError{Count: msg.Count})
		return
	}
	capacity := constants.GetBarracksTrainingCapacity(state.Building.Level)
	if msg.Count > capacity {
		ctx.Respond(&messages.TrainingCapacityExceededError{Requested: msg.Count, Capacity: capacity})
		return
	}

	popCost := msg.Count * constants.GetTroopPopCost(msg.Type)
	goldCost := msg.Count * constants.GetTroopGoldCost(msg.Type)

	// Reserve population first (a cheap cap check), then charge gold; roll the
	// reservation back if the owner can't afford it.
	res, err := state.Cluster.Request("city", state.Building.CityID, messages.ReserveMilitaryPopulationMessage{Count: popCost})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to reserve military population", "error", err)
		ctx.Respond(&messages.InternalError{})
		return
	}
	switch r := res.(type) {
	case messages.Ack:
		// reserved
	case *messages.InsufficientPopulationError:
		ctx.Respond(r)
		return
	default:
		ctx.Respond(&messages.InvalidResponseTypeError{})
		return
	}

	res, err = state.Cluster.Request("city", state.Building.CityID, messages.DeductOwnerGoldMessage{Amount: goldCost})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to deduct gold for training", "error", err)
		b.releasePopulation(state, popCost)
		ctx.Respond(&messages.InternalError{})
		return
	}
	switch r := res.(type) {
	case messages.Ack:
		// paid
	case messages.InsufficientGoldError:
		b.releasePopulation(state, popCost)
		ctx.Respond(&r)
		return
	default:
		b.releasePopulation(state, popCost)
		ctx.Respond(&messages.InvalidResponseTypeError{})
		return
	}

	b.queue = append(b.queue, trainingBatch{troopType: msg.Type, count: msg.Count})
	if len(b.queue) == 1 {
		b.startFront(ctx, state)
	}
	ctx.Respond(messages.Ack{})
}

func (b *barracksImpl) releasePopulation(state *buildingActor, count int64) {
	if err := state.Cluster.Tell("city", state.Building.CityID, messages.ReleaseMilitaryPopulationMessage{Count: count}); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to release military population on rollback", "error", err)
	}
}

func (b *barracksImpl) startFront(ctx actor.Context, state *buildingActor) {
	if len(b.queue) == 0 {
		return
	}
	trainTime := constants.GetTroopTrainTime(b.queue[0].troopType)
	b.queue[0].completeAt = time.Now().Add(time.Duration(trainTime) * time.Second)
	b.arm(ctx, b.queue[0].completeAt)
}

// arm schedules a PeriodicOperationMessage at the front batch's completion time.
func (b *barracksImpl) arm(ctx actor.Context, at time.Time) {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	delay := max(time.Until(at), 0)
	pid := ctx.Self()
	system := ctx.ActorSystem()
	b.timer = time.AfterFunc(delay, func() {
		system.Root.Send(pid, messages.PeriodicOperationMessage{})
	})
}

func (b *barracksImpl) checkComplete(ctx actor.Context, state *buildingActor) {
	if len(b.queue) == 0 {
		return
	}
	front := b.queue[0]
	if front.completeAt.IsZero() {
		// Front hasn't started (e.g. after a restart); kick it off.
		b.startFront(ctx, state)
		return
	}
	if time.Now().Before(front.completeAt) {
		return
	}
	b.spawnArmy(ctx, state, front)
	b.queue = b.queue[1:]
	if len(b.queue) > 0 {
		b.startFront(ctx, state)
	}
}

func (b *barracksImpl) spawnArmy(ctx actor.Context, state *buildingActor, batch trainingBatch) {
	owner := b.cityOwner(state)
	if owner == nil {
		slog.WarnContext(state.Ctx(), "barracks completed training with no city owner; discarding batch", "building_id", state.Building.BuildingID)
		return
	}
	armyID := uuid.New().String()
	if _, err := state.Cluster.Request("army", armyID, &messages.CreateArmyMessage{
		Army: domain.Army{
			ArmyID: armyID,
			Owner:  *owner,
			X:      state.Building.X,
			Y:      state.Building.Y,
			Troops: map[domain.TroopType]int64{batch.troopType: batch.count},
		},
	}); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to spawn army from training", "building_id", state.Building.BuildingID, "error", err)
		return
	}
	slog.InfoContext(state.Ctx(), "training complete, army spawned",
		"building_id", state.Building.BuildingID,
		"army_id", armyID,
		"troop_type", batch.troopType,
		"count", batch.count,
	)
}

func (b *barracksImpl) cityOwner(state *buildingActor) *string {
	res, err := state.Cluster.Request("city", state.Building.CityID, messages.GetCityMessage{})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to get city for army spawn", "error", err)
		return nil
	}
	resp, ok := res.(*messages.GetCityResponseMessage)
	if !ok {
		return nil
	}
	return resp.City.Owner
}
