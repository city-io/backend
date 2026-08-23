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

type barracksImpl struct {
	queue  []domain.TrainingOrder
	timer  *time.Timer
	loaded bool
}

func newBarracksImpl() buildingActorImpl {
	return &barracksImpl{}
}

func (b *barracksImpl) Create(ctx actor.Context, state *buildingActor) {
	b.loadQueue(ctx, state)
}

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
	case messages.GetTrainingOrdersMessage:
		if !b.loaded && !b.loadQueue(ctx, state) {
			ctx.Respond(&messages.InternalError{})
			return
		}
		ctx.Respond(&messages.GetTrainingOrdersResponseMessage{Orders: append([]domain.TrainingOrder(nil), b.queue...)})
	case messages.PeriodicOperationMessage:
		b.checkComplete(ctx, state)
	}
}

func (b *barracksImpl) loadQueue(ctx actor.Context, state *buildingActor) bool {
	orders, err := state.Store.GetTrainingOrdersByBarracks(state.Ctx(), state.Building.BuildingID)
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to restore barracks training queue", "building_id", state.Building.BuildingID, "error", err)
		return false
	}
	b.queue = orders
	b.loaded = true
	if len(b.queue) == 0 {
		return true
	}
	if b.queue[0].CompletesAt.Time == nil {
		b.startFront(ctx, state)
	} else {
		b.arm(ctx, *b.queue[0].CompletesAt.Time)
	}
	return true
}

func (b *barracksImpl) handleTrain(ctx actor.Context, state *buildingActor, msg messages.TrainTroopsMessage) {
	if !b.loaded && !b.loadQueue(ctx, state) {
		ctx.Respond(&messages.InternalError{})
		return
	}
	if state.constructionActive() {
		ctx.Respond(&messages.ConstructionInProgressError{BuildingID: state.Building.BuildingID})
		return
	}
	if msg.Count <= 0 {
		ctx.Respond(&messages.InvalidTroopCountError{Count: msg.Count})
		return
	}
	if !constants.IsValidTroopType(msg.Type) {
		ctx.Respond(&messages.InvalidTroopTypeError{Type: msg.Type})
		return
	}
	capacity := constants.GetBarracksTrainingCapacity(state.Building.Level)
	if msg.Count > capacity {
		ctx.Respond(&messages.TrainingCapacityExceededError{Requested: msg.Count, Capacity: capacity})
		return
	}

	popCost := msg.Count * constants.GetTroopPopCost(msg.Type)
	goldCost := msg.Count * constants.GetTroopGoldCost(msg.Type)
	if !b.recruitPopulation(ctx, state, popCost) {
		return
	}
	if !b.deductGold(ctx, state, goldCost) {
		b.releasePopulation(state, popCost)
		return
	}

	now := time.Now()
	order := domain.TrainingOrder{
		TrainingOrderID: uuid.New().String(),
		ArmyID:          uuid.New().String(),
		BarracksID:      state.Building.BuildingID,
		TroopType:       msg.Type,
		Count:           msg.Count,
		PopulationCost:  popCost,
		GoldCost:        goldCost,
		CreatedAt:       now,
	}
	if err := state.Store.CreateTrainingOrder(state.Ctx(), order); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to persist training order", "building_id", state.Building.BuildingID, "error", err)
		b.rollbackCost(state, order)
		ctx.Respond(&messages.InternalError{})
		return
	}

	b.queue = append(b.queue, order)
	if len(b.queue) == 1 {
		b.startFront(ctx, state)
	}
	ctx.Respond(&messages.TrainTroopsResponseMessage{Order: b.queue[len(b.queue)-1]})
}

func (b *barracksImpl) recruitPopulation(ctx actor.Context, state *buildingActor, count int64) bool {
	res, err := state.Cluster.Request("city", state.Building.CityID, messages.RecruitPopulationMessage{Count: count})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to recruit city population", "error", err)
		ctx.Respond(&messages.InternalError{})
		return false
	}
	switch response := res.(type) {
	case messages.Ack:
		return true
	case *messages.InsufficientPopulationError:
		ctx.Respond(response)
	default:
		ctx.Respond(&messages.InvalidResponseTypeError{})
	}
	return false
}

func (b *barracksImpl) deductGold(ctx actor.Context, state *buildingActor, amount int64) bool {
	res, err := state.Cluster.Request("city", state.Building.CityID, messages.DeductOwnerGoldMessage{Amount: amount})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to deduct gold for training", "error", err)
		ctx.Respond(&messages.InternalError{})
		return false
	}
	switch response := res.(type) {
	case messages.Ack:
		return true
	case messages.InsufficientGoldError:
		ctx.Respond(&response)
	default:
		ctx.Respond(&messages.InvalidResponseTypeError{})
	}
	return false
}

func (b *barracksImpl) rollbackCost(state *buildingActor, order domain.TrainingOrder) {
	b.releasePopulation(state, order.PopulationCost)
	res, err := state.Cluster.Request("city", state.Building.CityID, messages.CreditOwnerGoldMessage{Amount: order.GoldCost})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to refund training gold", "building_id", state.Building.BuildingID, "error", err)
		return
	}
	if _, ok := res.(messages.Ack); !ok {
		slog.ErrorContext(state.Ctx(), "training gold refund returned unexpected response", "building_id", state.Building.BuildingID)
	}
}

func (b *barracksImpl) releasePopulation(state *buildingActor, count int64) {
	res, err := state.Cluster.Request("city", state.Building.CityID, messages.ReturnRecruitsMessage{Count: count})
	if err != nil {
		slog.ErrorContext(state.Ctx(), "failed to return recruits on rollback", "error", err)
		return
	}
	if _, ok := res.(messages.Ack); !ok {
		slog.ErrorContext(state.Ctx(), "recruit rollback returned unexpected response", "building_id", state.Building.BuildingID)
	}
}

func (b *barracksImpl) startFront(ctx actor.Context, state *buildingActor) bool {
	if len(b.queue) == 0 {
		return true
	}
	now := time.Now()
	completeAt := now.Add(time.Duration(constants.GetTroopTrainingDuration(b.queue[0].TroopType, b.queue[0].Count)) * time.Second)
	if err := state.Store.StartTrainingOrder(state.Ctx(), b.queue[0].TrainingOrderID, now, completeAt); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to start training order", "training_order_id", b.queue[0].TrainingOrderID, "error", err)
		return false
	}
	b.queue[0].StartedAt = domain.NullTime{Time: &now}
	b.queue[0].CompletesAt = domain.NullTime{Time: &completeAt}
	b.arm(ctx, completeAt)
	return true
}

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
	if !b.loaded && !b.loadQueue(ctx, state) {
		return
	}
	if len(b.queue) == 0 {
		return
	}
	front := b.queue[0]
	if front.CompletesAt.Time == nil {
		b.startFront(ctx, state)
		return
	}
	if time.Now().Before(*front.CompletesAt.Time) {
		return
	}
	if !b.spawnArmy(ctx, state, front) {
		return
	}
	if err := state.Store.DeleteTrainingOrder(state.Ctx(), front.TrainingOrderID); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to finish training order", "training_order_id", front.TrainingOrderID, "error", err)
		return
	}
	b.queue = b.queue[1:]
	if len(b.queue) > 0 {
		b.startFront(ctx, state)
	}
}

func (b *barracksImpl) spawnArmy(ctx actor.Context, state *buildingActor, order domain.TrainingOrder) bool {
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
	if _, ok := res.(messages.Ack); !ok {
		slog.ErrorContext(state.Ctx(), "army spawn returned unexpected response", "building_id", state.Building.BuildingID)
		return false
	}
	slog.InfoContext(state.Ctx(), "training complete, army spawned",
		"building_id", state.Building.BuildingID,
		"army_id", order.ArmyID,
		"troop_type", order.TroopType,
		"count", order.Count,
	)
	return true
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
