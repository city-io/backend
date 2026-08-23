package actors

import (
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/battles"
	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/messages"
	"cityio/internal/stream"
)

type battleActor struct {
	baseActor
	Battle        domain.Battle
	ticker        *time.Ticker
	stop          chan struct{}
	casualtyCarry map[string]float64
}

func NewBattleActor() BaseActorInterface { return &battleActor{} }
func (*battleActor) ActorType() string   { return "battle" }

func (state *battleActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *messages.CreateBattleMessage:
		if state.Battle.BattleID == msg.Battle.BattleID {
			ctx.Respond(messages.Ack{})
			return
		}
		state.Battle = msg.Battle
		state.casualtyCarry = make(map[string]float64)
		for _, id := range append(append([]string{}, state.Battle.Attackers.ArmyIDs...), state.Battle.Defenders.ArmyIDs...) {
			if err := state.Cluster.Tell("army", id, messages.EnterBattleMessage{BattleID: state.Battle.BattleID}); err != nil {
				slog.WarnContext(state.Ctx(), "failed to enroll army in battle", "battle_id", state.Battle.BattleID, "army_id", id, "error", err)
			}
		}
		state.publish()
		state.start(ctx)
		ctx.Respond(messages.Ack{})
	case messages.GetBattleMessage:
		ctx.Respond(&messages.GetBattleResponseMessage{Battle: state.Battle})
	case messages.RetreatFromBattleMessage:
		state.removeArmy(msg.ArmyID)
		state.publish()
		state.resolveIfFinished(ctx)
		ctx.Respond(messages.Ack{})
	case messages.JoinBattleMessage:
		if !state.join(msg) {
			ctx.Respond(&messages.InternalError{})
			return
		}
		state.publish()
		ctx.Respond(messages.Ack{})
	case messages.PeriodicOperationMessage:
		state.tick(ctx)
	}
}

func (state *battleActor) join(msg messages.JoinBattleMessage) bool {
	if state.Battle.Defenders.GarrisonCityID != nil && *state.Battle.Defenders.GarrisonCityID == msg.OpposesGarrisonCityID {
		state.Battle.Attackers.UserIDs = appendUnique(state.Battle.Attackers.UserIDs, msg.Owner)
		state.Battle.Attackers.ArmyIDs = appendUnique(state.Battle.Attackers.ArmyIDs, msg.ArmyID)
		return true
	}
	if state.Battle.Attackers.GarrisonCityID != nil && *state.Battle.Attackers.GarrisonCityID == msg.OpposesGarrisonCityID {
		state.Battle.Defenders.UserIDs = appendUnique(state.Battle.Defenders.UserIDs, msg.Owner)
		state.Battle.Defenders.ArmyIDs = appendUnique(state.Battle.Defenders.ArmyIDs, msg.ArmyID)
		return true
	}
	if contains(state.Battle.Defenders.ArmyIDs, msg.OpposesArmyID) {
		state.Battle.Attackers.UserIDs = appendUnique(state.Battle.Attackers.UserIDs, msg.Owner)
		state.Battle.Attackers.ArmyIDs = appendUnique(state.Battle.Attackers.ArmyIDs, msg.ArmyID)
		return true
	}
	if contains(state.Battle.Attackers.ArmyIDs, msg.OpposesArmyID) {
		state.Battle.Defenders.UserIDs = appendUnique(state.Battle.Defenders.UserIDs, msg.Owner)
		state.Battle.Defenders.ArmyIDs = appendUnique(state.Battle.Defenders.ArmyIDs, msg.ArmyID)
		return true
	}
	return false
}

type battleArmy struct {
	id   string
	army domain.Army
}

func (state *battleActor) armies(ids []string) []battleArmy {
	result := make([]battleArmy, 0, len(ids))
	for _, id := range ids {
		res, err := state.Cluster.Request("army", id, messages.GetArmyMessage{})
		if err != nil {
			continue
		}
		response, ok := res.(*messages.GetArmyResponseMessage)
		if !ok || response.Army.BattleID == nil || *response.Army.BattleID != state.Battle.BattleID {
			continue
		}
		result = append(result, battleArmy{id: id, army: response.Army})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].id < result[j].id })
	return result
}

func attackPower(armies []battleArmy, garrisonCount int64) float64 {
	var power float64
	for _, participant := range armies {
		for troopType, count := range participant.army.Troops {
			power += float64(count * constants.GetTroopStat(troopType).Attack)
		}
	}
	power += float64(garrisonCount * constants.GetTroopStat(domain.TroopTypeSoldier).Attack)
	return power
}

func (state *battleActor) casualties(targets []battleArmy, garrisonCityID *string, garrisonCount int64, incoming float64) (map[string]map[domain.TroopType]int64, int64) {
	result := make(map[string]map[domain.TroopType]int64)
	var durability float64
	for _, target := range targets {
		for troopType, count := range target.army.Troops {
			stat := constants.GetTroopStat(troopType)
			durability += float64(count * (stat.HP + 5*stat.Defense))
		}
	}
	if garrisonCount > 0 {
		stat := constants.GetTroopStat(domain.TroopTypeSoldier)
		durability += float64(garrisonCount * (stat.HP + 5*stat.Defense))
	}
	if durability <= 0 || incoming <= 0 {
		return result, 0
	}
	for _, target := range targets {
		for _, troopType := range constants.AllTroopTypes() {
			count := target.army.Troops[troopType]
			if count <= 0 {
				continue
			}
			key := fmt.Sprintf("%s:%s", target.id, troopType)
			expected := incoming*float64(count)/durability + state.casualtyCarry[key]
			kills := min(count, int64(math.Floor(expected)))
			state.casualtyCarry[key] = expected - float64(kills)
			if kills > 0 {
				if result[target.id] == nil {
					result[target.id] = make(map[domain.TroopType]int64)
				}
				result[target.id][troopType] = kills
			}
		}
	}
	var garrisonKills int64
	if garrisonCityID != nil && garrisonCount > 0 {
		key := "garrison:" + *garrisonCityID
		expected := incoming*float64(garrisonCount)/durability + state.casualtyCarry[key]
		garrisonKills = min(garrisonCount, int64(math.Floor(expected)))
		state.casualtyCarry[key] = expected - float64(garrisonKills)
	}
	return result, garrisonKills
}

func (state *battleActor) tick(ctx actor.Context) {
	attackers := state.armies(state.Battle.Attackers.ArmyIDs)
	defenders := state.armies(state.Battle.Defenders.ArmyIDs)
	state.Battle.Attackers.ArmyIDs = armyIDs(attackers)
	state.Battle.Defenders.ArmyIDs = armyIDs(defenders)
	state.refreshGarrison(&state.Battle.Attackers)
	state.refreshGarrison(&state.Battle.Defenders)
	if state.resolveIfFinished(ctx) {
		return
	}
	toDefenders, defenderGarrisonLosses := state.casualties(defenders, state.Battle.Defenders.GarrisonCityID, state.Battle.Defenders.GarrisonCount, attackPower(attackers, state.Battle.Attackers.GarrisonCount))
	toAttackers, attackerGarrisonLosses := state.casualties(attackers, state.Battle.Attackers.GarrisonCityID, state.Battle.Attackers.GarrisonCount, attackPower(defenders, state.Battle.Defenders.GarrisonCount))
	for armyID, casualties := range toDefenders {
		if !state.apply(armyID, casualties) {
			state.removeArmy(armyID)
		}
	}
	for armyID, casualties := range toAttackers {
		if !state.apply(armyID, casualties) {
			state.removeArmy(armyID)
		}
	}
	state.applyGarrisonLosses(&state.Battle.Defenders, defenderGarrisonLosses)
	state.applyGarrisonLosses(&state.Battle.Attackers, attackerGarrisonLosses)
	if state.resolveIfFinished(ctx) {
		return
	}
	state.Battle.NextTick = time.Now().Add(constants.BattleTickInterval)
	state.publish()
}

func (state *battleActor) refreshGarrison(side *domain.BattleSide) {
	if side.GarrisonCityID == nil {
		side.GarrisonCount = 0
		return
	}
	res, err := state.Cluster.Request("city", *side.GarrisonCityID, messages.GetCityMessage{})
	if err != nil {
		side.GarrisonCount = 0
		return
	}
	response, ok := res.(*messages.GetCityResponseMessage)
	if !ok {
		side.GarrisonCount = 0
		return
	}
	side.GarrisonCount = int64(math.Floor(response.City.GarrisonPopulation))
}

func (state *battleActor) applyGarrisonLosses(side *domain.BattleSide, count int64) {
	if side.GarrisonCityID == nil || count <= 0 {
		return
	}
	res, err := state.Cluster.Request("city", *side.GarrisonCityID, messages.ApplyGarrisonCasualtiesMessage{Count: count})
	if err != nil {
		slog.WarnContext(state.Ctx(), "failed to apply garrison casualties", "battle_id", state.Battle.BattleID, "city_id", *side.GarrisonCityID, "error", err)
		return
	}
	if response, ok := res.(*messages.ApplyGarrisonCasualtiesResponseMessage); ok && !response.Survived {
		side.GarrisonCount = 0
	} else {
		side.GarrisonCount = max(side.GarrisonCount-count, 0)
	}
}

func (state *battleActor) apply(armyID string, casualties map[domain.TroopType]int64) bool {
	res, err := state.Cluster.Request("army", armyID, messages.ApplyCasualtiesMessage{Casualties: casualties})
	if err != nil {
		slog.WarnContext(state.Ctx(), "failed to apply battle casualties", "battle_id", state.Battle.BattleID, "army_id", armyID, "error", err)
		return true
	}
	response, ok := res.(*messages.ApplyCasualtiesResponseMessage)
	return !ok || response.Survived
}

func armyIDs(armies []battleArmy) []string {
	result := make([]string, 0, len(armies))
	for _, army := range armies {
		result = append(result, army.id)
	}
	return result
}

func (state *battleActor) removeArmy(id string) {
	state.Battle.Attackers.ArmyIDs = removeID(state.Battle.Attackers.ArmyIDs, id)
	state.Battle.Defenders.ArmyIDs = removeID(state.Battle.Defenders.ArmyIDs, id)
}

func removeID(ids []string, remove string) []string {
	result := ids[:0]
	for _, id := range ids {
		if id != remove {
			result = append(result, id)
		}
	}
	return result
}

func contains(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

func appendUnique(ids []string, id string) []string {
	if contains(ids, id) {
		return ids
	}
	return append(ids, id)
}

func (state *battleActor) resolveIfFinished(ctx actor.Context) bool {
	attackersActive := len(state.Battle.Attackers.ArmyIDs) > 0 || state.Battle.Attackers.GarrisonCount > 0
	defendersActive := len(state.Battle.Defenders.ArmyIDs) > 0 || state.Battle.Defenders.GarrisonCount > 0
	if attackersActive && defendersActive {
		return false
	}
	winners := state.Battle.Attackers.ArmyIDs
	if !attackersActive {
		winners = state.Battle.Defenders.ArmyIDs
	}
	state.endGarrisonBattle(state.Battle.Attackers.GarrisonCityID)
	state.endGarrisonBattle(state.Battle.Defenders.GarrisonCityID)
	for _, id := range winners {
		_ = state.Cluster.Tell("army", id, messages.LeaveBattleMessage{BattleID: state.Battle.BattleID})
	}
	state.stopTicker()
	battles.Delete(state.Battle.BattleID)
	id := state.Battle.BattleID
	stream.Publish("", stream.StateUpdate{DeletedBattleID: &id})
	ctx.Stop(ctx.Self())
	return true
}

func (state *battleActor) endGarrisonBattle(cityID *string) {
	if cityID != nil {
		_ = state.Cluster.Tell("city", *cityID, messages.EndGarrisonBattleMessage{BattleID: state.Battle.BattleID})
	}
}

func (state *battleActor) publish() {
	battles.Upsert(state.Battle)
	b := state.Battle
	stream.Publish("", stream.StateUpdate{Battle: &b})
}

func (state *battleActor) start(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.BattleTickInterval)
	state.stop = make(chan struct{})
	pid, system := ctx.Self(), ctx.ActorSystem()
	go func() {
		for {
			select {
			case <-state.ticker.C:
				system.Root.Send(pid, messages.PeriodicOperationMessage{})
			case <-state.stop:
				state.ticker.Stop()
				return
			}
		}
	}()
}

func (state *battleActor) stopTicker() {
	if state.stop == nil {
		return
	}
	select {
	case <-state.stop:
	default:
		close(state.stop)
	}
}
