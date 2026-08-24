package actors

import (
	"log/slog"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/domain"
	"cityio/internal/messages"
)

// barracksImpl is one worker lane in its city's shared training pipeline. It
// owns only the assigned order and completion timer; queued orders belong to
// the city actor.
type barracksImpl struct {
	active *domain.TrainingOrder
	timer  *time.Timer
}

func newBarracksImpl() buildingActorImpl { return &barracksImpl{} }

func (b *barracksImpl) Create(_ actor.Context, state *buildingActor) {
	_ = state.Cluster.Tell("city", state.Building.CityID, messages.RegisterBarracksMessage{BarracksID: state.Building.BuildingID})
}

func (b *barracksImpl) Destroy(_ actor.Context, state *buildingActor) {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	_ = state.Cluster.Tell("city", state.Building.CityID, messages.UnregisterBarracksMessage{BarracksID: state.Building.BuildingID})
}

func (b *barracksImpl) Handle(ctx actor.Context, state *buildingActor) {
	switch ctx.Message().(type) {
	case messages.TrainingQueueAvailableMessage:
		b.requestWork(ctx, state)
	case messages.PeriodicOperationMessage:
		b.checkComplete(ctx, state)
		b.requestWork(ctx, state)
	}
}

func (b *barracksImpl) requestWork(ctx actor.Context, state *buildingActor) bool {
	if b.active != nil || state.constructionActive() || state.Building.Level <= 0 {
		return true
	}
	res, err := state.Cluster.Request("city", state.Building.CityID, messages.ClaimTrainingOrderMessage{
		BarracksID: state.Building.BuildingID,
		Level:      state.Building.Level,
	})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to claim city training order", "building_id", state.Building.BuildingID, "error", err)
		return false
	}
	response, ok := res.(*messages.ClaimTrainingOrderResponseMessage)
	if !ok {
		return false
	}
	if response.Order == nil {
		return true
	}
	b.active = response.Order
	if b.active.CompletesAt.Time != nil {
		b.arm(ctx, *b.active.CompletesAt.Time)
	}
	return true
}

func (b *barracksImpl) ensureInitialized(ctx actor.Context, state *buildingActor) bool {
	// A completed idle barracks must check the city queue before it can be
	// upgraded or demolished. This closes the window between an asynchronous
	// queue notification and the building command reaching this actor.
	return b.requestWork(ctx, state)
}

func (b *barracksImpl) arm(ctx actor.Context, at time.Time) {
	if b.timer != nil {
		b.timer.Stop()
	}
	delay := max(time.Until(at), 0)
	pid, system := ctx.Self(), ctx.ActorSystem()
	b.timer = time.AfterFunc(delay, func() {
		system.Root.Send(pid, messages.PeriodicOperationMessage{})
	})
}

func (b *barracksImpl) checkComplete(ctx actor.Context, state *buildingActor) {
	if b.active == nil || b.active.CompletesAt.Time == nil || time.Now().Before(*b.active.CompletesAt.Time) {
		return
	}
	order := *b.active
	if !b.spawnArmy(state, order) {
		return
	}
	res, err := state.Cluster.Request("city", state.Building.CityID, messages.CompleteTrainingOrderMessage{
		TrainingOrderID: order.TrainingOrderID,
		BarracksID:      state.Building.BuildingID,
	})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to finish city training order", "building_id", state.Building.BuildingID, "training_order_id", order.TrainingOrderID, "error", err)
		return
	}
	if _, ok := res.(messages.Ack); !ok {
		return
	}
	b.active = nil
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	slog.InfoContext(state.Ctx(), "training complete, army spawned",
		"building_id", state.Building.BuildingID,
		"army_id", order.ArmyID,
		"troop_type", order.TroopType,
		"count", order.Count,
	)
	b.requestWork(ctx, state)
}

func (b *barracksImpl) spawnArmy(state *buildingActor, order domain.TrainingOrder) bool {
	owner := b.cityOwner(state)
	if owner == nil {
		slog.WarnContext(state.Ctx(), "barracks completed training with no city owner", "building_id", state.Building.BuildingID)
		return false
	}
	res, err := state.Cluster.Request("army", order.ArmyID, &messages.CreateArmyMessage{
		Army: domain.Army{
			ArmyID: order.ArmyID,
			Owner:  *owner,
			X:      state.Building.X,
			Y:      state.Building.Y,
			Troops: map[domain.TroopType]int64{order.TroopType: order.Count},
		},
	})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to spawn army from training", "building_id", state.Building.BuildingID, "error", err)
		return false
	}
	_, ok := res.(messages.Ack)
	return ok
}

func (b *barracksImpl) cityOwner(state *buildingActor) *string {
	res, err := state.Cluster.Request("city", state.Building.CityID, messages.GetCityMessage{})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to get city for army spawn", "error", err)
		return nil
	}
	response, ok := res.(*messages.GetCityResponseMessage)
	if !ok {
		return nil
	}
	return response.City.Owner
}
