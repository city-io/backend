package actors

import (
	"crypto/sha256"
	"log/slog"
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"

	"cityio/internal/battles"
	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/messages"
	"cityio/internal/stream"
	"cityio/internal/utils"
)

type battleActor struct {
	baseActor
	Battle                domain.Battle
	roundTimer            *time.Timer
	casualtyRNG           *rand.Rand
	civilianCasualtyCarry map[string]float64
	reportAttackers       domain.BattleReportSide
	reportDefenders       domain.BattleReportSide
	reportRounds          []domain.BattleReportRound
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
		state.casualtyRNG = newBattleCasualtyRNG(state.Battle.BattleID)
		state.civilianCasualtyCarry = make(map[string]float64)
		armySnapshots := make(map[string]domain.Army, len(msg.Armies))
		for _, army := range msg.Armies {
			armySnapshots[army.ArmyID] = army
		}
		state.reportAttackers = state.snapshotReportSide(state.Battle.Attackers, armySnapshots)
		state.reportDefenders = state.snapshotReportSide(state.Battle.Defenders, armySnapshots)
		state.reportRounds = nil
		for _, id := range append(append([]string{}, state.Battle.Attackers.ArmyIDs...), state.Battle.Defenders.ArmyIDs...) {
			if err := state.Cluster.Tell("army", id, messages.EnterBattleMessage{BattleID: state.Battle.BattleID}); err != nil {
				slog.WarnContext(state.Ctx(), "failed to enroll army in battle", "battle_id", state.Battle.BattleID, "army_id", id, "error", err)
			}
		}
		state.updateDefenseBonuses()
		state.armNextRound(ctx)
		state.publish()
		ctx.Respond(messages.Ack{})
	case messages.GetBattleMessage:
		state.syncLiveSummary()
		ctx.Respond(&messages.GetBattleResponseMessage{Battle: state.Battle})
	case messages.RetreatFromBattleMessage:
		state.markRetreated(msg.ArmyID)
		state.removeArmy(msg.ArmyID)
		state.publish()
		state.resolveIfFinished(ctx, domain.BattleReportResolutionRetreat)
		ctx.Respond(messages.Ack{})
	case messages.JoinBattleMessage:
		if !state.join(msg) {
			ctx.Respond(&messages.InternalError{})
			return
		}
		state.recordJoinedArmy(msg.Army)
		state.updateDefenseBonuses()
		state.publish()
		ctx.Respond(messages.Ack{})
	case messages.PeriodicOperationMessage:
		state.tick(ctx)
	}
}

func (state *battleActor) join(msg messages.JoinBattleMessage) bool {
	armyID, owner := msg.Army.ArmyID, msg.Army.Owner
	if state.Battle.Defenders.MilitiaCityID != nil && *state.Battle.Defenders.MilitiaCityID == msg.OpposesMilitiaCityID {
		state.Battle.Attackers.UserIDs = appendUnique(state.Battle.Attackers.UserIDs, owner)
		state.Battle.Attackers.ArmyIDs = appendUnique(state.Battle.Attackers.ArmyIDs, armyID)
		return true
	}
	if state.Battle.Attackers.MilitiaCityID != nil && *state.Battle.Attackers.MilitiaCityID == msg.OpposesMilitiaCityID {
		state.Battle.Defenders.UserIDs = appendUnique(state.Battle.Defenders.UserIDs, owner)
		state.Battle.Defenders.ArmyIDs = appendUnique(state.Battle.Defenders.ArmyIDs, armyID)
		return true
	}
	if contains(state.Battle.Defenders.ArmyIDs, msg.OpposesArmyID) {
		state.Battle.Attackers.UserIDs = appendUnique(state.Battle.Attackers.UserIDs, owner)
		state.Battle.Attackers.ArmyIDs = appendUnique(state.Battle.Attackers.ArmyIDs, armyID)
		return true
	}
	if contains(state.Battle.Attackers.ArmyIDs, msg.OpposesArmyID) {
		state.Battle.Defenders.UserIDs = appendUnique(state.Battle.Defenders.UserIDs, owner)
		state.Battle.Defenders.ArmyIDs = appendUnique(state.Battle.Defenders.ArmyIDs, armyID)
		return true
	}
	return false
}

func (state *battleActor) snapshotReportSide(side domain.BattleSide, armies map[string]domain.Army) domain.BattleReportSide {
	report := domain.BattleReportSide{
		UserIDs:          append([]string(nil), side.UserIDs...),
		MilitiaCityID:    side.MilitiaCityID,
		StartingMilitia:  side.MilitiaCount,
		SurvivingMilitia: side.MilitiaCount,
	}
	state.snapshotCommanders(&report)
	state.snapshotSettlement(&report)
	for _, armyID := range side.ArmyIDs {
		if army, ok := armies[armyID]; ok {
			state.addArmyToReport(&report, army)
		}
	}
	return report
}

func (state *battleActor) addArmyToReport(report *domain.BattleReportSide, army domain.Army) {
	report.Armies = append(report.Armies, domain.BattleReportArmy{
		ArmyID:          army.ArmyID,
		Name:            army.Name,
		OwnerID:         army.Owner,
		StartingTroops:  cloneTroops(army.Troops),
		SurvivingTroops: cloneTroops(army.Troops),
	})
}

func (state *battleActor) recordJoinedArmy(army domain.Army) {
	armyID := army.ArmyID
	if contains(state.Battle.Attackers.ArmyIDs, armyID) && state.reportArmy(&state.reportAttackers, armyID) == nil {
		state.reportAttackers.UserIDs = append([]string(nil), state.Battle.Attackers.UserIDs...)
		state.snapshotCommanders(&state.reportAttackers)
		state.addArmyToReport(&state.reportAttackers, army)
	}
	if contains(state.Battle.Defenders.ArmyIDs, armyID) && state.reportArmy(&state.reportDefenders, armyID) == nil {
		state.reportDefenders.UserIDs = append([]string(nil), state.Battle.Defenders.UserIDs...)
		state.snapshotCommanders(&state.reportDefenders)
		state.addArmyToReport(&state.reportDefenders, army)
	}
}

func (state *battleActor) snapshotCommanders(side *domain.BattleReportSide) {
	for _, userID := range side.UserIDs {
		found := false
		for _, commander := range side.Commanders {
			if commander.UserID == userID {
				found = true
				break
			}
		}
		if found {
			continue
		}
		commander := domain.BattleReportCommander{UserID: userID}
		if user, err := state.Store.GetUserByID(state.Ctx(), userID); err == nil {
			commander.Username = user.Username
		}
		side.Commanders = append(side.Commanders, commander)
	}
}

func (state *battleActor) snapshotSettlement(side *domain.BattleReportSide) {
	if side.MilitiaCityID == nil {
		return
	}
	res, err := state.Cluster.Request("city", *side.MilitiaCityID, messages.GetCityMessage{})
	if err != nil {
		return
	}
	response, ok := res.(*messages.GetCityResponseMessage)
	if !ok {
		return
	}
	city := response.City
	side.Settlement = &domain.BattleReportSettlement{
		CityID:             city.CityID,
		Name:               city.Name,
		Type:               city.Type,
		OwnerID:            city.Owner,
		StartingPopulation: city.Population,
		EndingPopulation:   city.Population,
	}
}

func (state *battleActor) refreshReportSettlement(side *domain.BattleReportSide) {
	if side.Settlement == nil {
		return
	}
	res, err := state.Cluster.Request("city", side.Settlement.CityID, messages.GetCityMessage{})
	if err != nil {
		return
	}
	if response, ok := res.(*messages.GetCityResponseMessage); ok {
		side.Settlement.EndingPopulation = response.City.Population
	}
}

func cloneTroops(troops map[domain.TroopType]int64) map[domain.TroopType]int64 {
	result := make(map[domain.TroopType]int64, len(troops))
	for troopType, count := range troops {
		result[troopType] = count
	}
	return result
}

func (state *battleActor) reportArmy(side *domain.BattleReportSide, armyID string) *domain.BattleReportArmy {
	for i := range side.Armies {
		if side.Armies[i].ArmyID == armyID {
			return &side.Armies[i]
		}
	}
	return nil
}

func (state *battleActor) markRetreated(armyID string) {
	for _, side := range []*domain.BattleReportSide{&state.reportAttackers, &state.reportDefenders} {
		if army := state.reportArmy(side, armyID); army != nil {
			army.Retreated = true
			return
		}
	}
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

func attackPower(armies []battleArmy, militiaCount int64) float64 {
	var power float64
	for _, participant := range armies {
		for troopType, count := range participant.army.Troops {
			power += float64(count * constants.GetTroopStat(troopType).Attack)
		}
	}
	power += float64(militiaCount * constants.GetTroopStat(domain.TroopTypeSoldier).Attack)
	return power
}

func newBattleCasualtyRNG(battleID string) *rand.Rand {
	seed := sha256.Sum256([]byte(battleID))
	return rand.New(rand.NewChaCha8(seed))
}

func (state *battleActor) rollCasualties(count int64, probability float64) int64 {
	if count <= 0 || probability <= 0 {
		return 0
	}
	if probability >= 1 {
		return count
	}
	if state.casualtyRNG == nil {
		state.casualtyRNG = newBattleCasualtyRNG(state.Battle.BattleID)
	}
	var casualties int64
	for range count {
		if state.casualtyRNG.Float64() < probability {
			casualties++
		}
	}
	return casualties
}

func (state *battleActor) casualties(targets []battleArmy, militiaCityID *string, militiaCount int64, incoming float64, defenseBonusPercent int) (map[string]map[domain.TroopType]int64, int64) {
	result := make(map[string]map[domain.TroopType]int64)
	var durability float64
	for _, target := range targets {
		for troopType, count := range target.army.Troops {
			stat := constants.GetTroopStat(troopType)
			durability += float64(count * (stat.HP + 5*stat.Defense))
		}
	}
	if militiaCount > 0 {
		stat := constants.GetTroopStat(domain.TroopTypeSoldier)
		durability += float64(militiaCount * (stat.HP + 5*stat.Defense))
	}
	durability *= 1 + float64(max(defenseBonusPercent, 0))/100
	if durability <= 0 || incoming <= 0 {
		return result, 0
	}
	// Every unit independently rolls the same casualty probability. This keeps
	// the existing expected losses while allowing smaller forces to occasionally
	// outperform the average and making larger battles proportionally steadier.
	probability := min(incoming*constants.BattleCasualtyRate/durability, 1)
	for _, target := range targets {
		for _, troopType := range constants.AllTroopTypes() {
			count := target.army.Troops[troopType]
			if count <= 0 {
				continue
			}
			kills := state.rollCasualties(count, probability)
			if kills > 0 {
				if result[target.id] == nil {
					result[target.id] = make(map[domain.TroopType]int64)
				}
				result[target.id][troopType] = kills
			}
		}
	}
	var militiaKills int64
	if militiaCityID != nil && militiaCount > 0 {
		militiaKills = state.rollCasualties(militiaCount, probability)
	}
	return result, militiaKills
}

func (state *battleActor) tick(ctx actor.Context) {
	attackers := state.armies(state.Battle.Attackers.ArmyIDs)
	defenders := state.armies(state.Battle.Defenders.ArmyIDs)
	state.Battle.Attackers.ArmyIDs = armyIDs(attackers)
	state.Battle.Defenders.ArmyIDs = armyIDs(defenders)
	state.refreshMilitia(&state.Battle.Attackers)
	state.refreshMilitia(&state.Battle.Defenders)
	if state.resolveIfFinished(ctx, domain.BattleReportResolutionElimination) {
		return
	}
	attackerPower := attackPower(attackers, state.Battle.Attackers.MilitiaCount)
	defenderPower := attackPower(defenders, state.Battle.Defenders.MilitiaCount)
	state.updateDefenseBonuses()
	toDefenders, defenderMilitiaLosses := state.casualties(defenders, state.Battle.Defenders.MilitiaCityID, state.Battle.Defenders.MilitiaCount, attackerPower, state.Battle.Defenders.DefenseBonusPercent)
	toAttackers, attackerMilitiaLosses := state.casualties(attackers, state.Battle.Attackers.MilitiaCityID, state.Battle.Attackers.MilitiaCount, defenderPower, state.Battle.Attackers.DefenseBonusPercent)
	defenderCivilianCasualties := state.applySiegeCivilianCasualties(&state.Battle.Defenders, militaryCasualtyCount(toDefenders, defenderMilitiaLosses))
	attackerCivilianCasualties := state.applySiegeCivilianCasualties(&state.Battle.Attackers, militaryCasualtyCount(toAttackers, attackerMilitiaLosses))
	state.reportRounds = append(state.reportRounds, domain.BattleReportRound{
		Number:                     len(state.reportRounds) + 1,
		OccurredAt:                 time.Now(),
		AttackerPower:              attackerPower,
		DefenderPower:              defenderPower,
		AttackerLosses:             reportLosses(toAttackers, state.Battle.Attackers.MilitiaCityID, attackerMilitiaLosses),
		DefenderLosses:             reportLosses(toDefenders, state.Battle.Defenders.MilitiaCityID, defenderMilitiaLosses),
		AttackerCivilianCasualties: attackerCivilianCasualties,
		DefenderCivilianCasualties: defenderCivilianCasualties,
	})
	state.recordCivilianCasualties(&state.reportDefenders, defenderCivilianCasualties)
	state.recordCivilianCasualties(&state.reportAttackers, attackerCivilianCasualties)
	for armyID, casualties := range toDefenders {
		state.recordTroopLosses(&state.reportDefenders, armyID, casualties)
		if !state.apply(armyID, casualties) {
			state.markDestroyed(&state.reportDefenders, armyID)
			state.removeArmy(armyID)
		}
	}
	for armyID, casualties := range toAttackers {
		state.recordTroopLosses(&state.reportAttackers, armyID, casualties)
		if !state.apply(armyID, casualties) {
			state.markDestroyed(&state.reportAttackers, armyID)
			state.removeArmy(armyID)
		}
	}
	state.recordMilitiaLosses(&state.reportDefenders, defenderMilitiaLosses)
	state.recordMilitiaLosses(&state.reportAttackers, attackerMilitiaLosses)
	state.applyMilitiaLosses(&state.Battle.Defenders, defenderMilitiaLosses)
	state.applyMilitiaLosses(&state.Battle.Attackers, attackerMilitiaLosses)
	if state.resolveIfFinished(ctx, domain.BattleReportResolutionElimination) {
		return
	}
	state.armNextRound(ctx)
	state.publish()
}

func (state *battleActor) updateDefenseBonuses() {
	fort := state.fortAtBattleTile()
	settlement := state.settlementAtBattleTile()
	state.Battle.Attackers.DefenseBonusPercent = state.defenseBonusPercent(state.Battle.Attackers, fort, settlement)
	state.Battle.Defenders.DefenseBonusPercent = state.defenseBonusPercent(state.Battle.Defenders, fort, settlement)
}

func (state *battleActor) fortAtBattleTile() *domain.Building {
	res, err := state.Cluster.Request("tile", utils.GetTileIndex(state.Battle.X, state.Battle.Y), messages.GetTileMessage{})
	if err != nil {
		return nil
	}
	tile, ok := res.(messages.GetTileResponseMessage)
	if !ok || tile.BuildingID == nil {
		return nil
	}
	res, err = state.Cluster.Request("building", *tile.BuildingID, messages.GetBuildingMessage{})
	if err != nil {
		return nil
	}
	building, ok := res.(*messages.GetBuildingResponseMessage)
	if !ok || building.Building.BuildingType() != domain.BuildingTypeFort || building.Building.Level <= 0 {
		return nil
	}
	return &building.Building
}

func (state *battleActor) settlementAtBattleTile() *domain.City {
	res, err := state.Cluster.Request("tile", utils.GetTileIndex(state.Battle.X, state.Battle.Y), messages.GetTileMessage{})
	if err != nil {
		return nil
	}
	tile, ok := res.(messages.GetTileResponseMessage)
	if !ok || tile.CityID == nil {
		return nil
	}
	res, err = state.Cluster.Request("city", *tile.CityID, messages.GetCityMessage{})
	if err != nil {
		return nil
	}
	city, ok := res.(*messages.GetCityResponseMessage)
	if !ok {
		return nil
	}
	return &city.City
}

func (state *battleActor) defenseBonusPercent(side domain.BattleSide, fort *domain.Building, settlement *domain.City) int {
	bonus := 0
	defendedCityID := side.MilitiaCityID
	if defendedCityID == nil && settlement != nil && settlement.Owner != nil && contains(side.UserIDs, *settlement.Owner) {
		defendedCityID = &settlement.CityID
	}
	if defendedCityID != nil {
		if buildings, err := state.Store.GetBuildingsByCity(state.Ctx(), *defendedCityID); err == nil {
			for _, building := range buildings {
				if res, requestErr := state.Cluster.Request("building", building.BuildingID, messages.GetBuildingMessage{}); requestErr == nil {
					if response, ok := res.(*messages.GetBuildingResponseMessage); ok {
						building = response.Building
					}
				}
				switch building.BuildingType() {
				case domain.BuildingTypeCityCenter, domain.BuildingTypeTownCenter:
					bonus += constants.GetBuildingDefenseBonusPercent(building.BuildingType(), building.Level)
				}
			}
		}
	}
	if fort != nil && fort.Owner != "" && contains(side.UserIDs, fort.Owner) {
		bonus += constants.GetBuildingDefenseBonusPercent(domain.BuildingTypeFort, fort.Level)
	}
	return bonus
}

func (state *battleActor) recordTroopLosses(side *domain.BattleReportSide, armyID string, casualties map[domain.TroopType]int64) {
	army := state.reportArmy(side, armyID)
	if army == nil {
		return
	}
	for troopType, count := range casualties {
		army.SurvivingTroops[troopType] = max(army.SurvivingTroops[troopType]-count, 0)
	}
}

func (state *battleActor) markDestroyed(side *domain.BattleReportSide, armyID string) {
	if army := state.reportArmy(side, armyID); army != nil {
		army.Destroyed = true
	}
}

func reportLosses(losses map[string]map[domain.TroopType]int64, militiaCityID *string, militia int64) []domain.BattleReportLoss {
	ids := make([]string, 0, len(losses))
	for armyID := range losses {
		ids = append(ids, armyID)
	}
	sort.Strings(ids)
	result := make([]domain.BattleReportLoss, 0, len(ids)+1)
	for _, armyID := range ids {
		id := armyID
		result = append(result, domain.BattleReportLoss{ArmyID: &id, Troops: cloneTroops(losses[armyID])})
	}
	if militiaCityID != nil && militia > 0 {
		result = append(result, domain.BattleReportLoss{MilitiaCityID: militiaCityID, Militia: militia})
	}
	return result
}

func militaryCasualtyCount(losses map[string]map[domain.TroopType]int64, militia int64) int64 {
	total := militia
	for _, armyLosses := range losses {
		for _, count := range armyLosses {
			total += count
		}
	}
	return total
}

func (state *battleActor) recordMilitiaLosses(side *domain.BattleReportSide, count int64) {
	side.SurvivingMilitia = max(side.SurvivingMilitia-count, 0)
}

func (state *battleActor) refreshMilitia(side *domain.BattleSide) {
	if side.MilitiaCityID == nil {
		side.MilitiaCount = 0
		return
	}
	res, err := state.Cluster.Request("city", *side.MilitiaCityID, messages.GetCityMessage{})
	if err != nil {
		side.MilitiaCount = 0
		return
	}
	response, ok := res.(*messages.GetCityResponseMessage)
	if !ok {
		side.MilitiaCount = 0
		return
	}
	side.MilitiaCount = int64(math.Floor(response.City.MilitiaPopulation))
}

func (state *battleActor) applyMilitiaLosses(side *domain.BattleSide, count int64) {
	if side.MilitiaCityID == nil || count <= 0 {
		return
	}
	res, err := state.Cluster.Request("city", *side.MilitiaCityID, messages.ApplyMilitiaCasualtiesMessage{Count: count})
	if err != nil {
		slog.WarnContext(state.Ctx(), "failed to apply militia casualties", "battle_id", state.Battle.BattleID, "city_id", *side.MilitiaCityID, "error", err)
		return
	}
	if response, ok := res.(*messages.ApplyMilitiaCasualtiesResponseMessage); ok && !response.Survived {
		side.MilitiaCount = 0
	} else {
		side.MilitiaCount = max(side.MilitiaCount-count, 0)
	}
}

func (state *battleActor) applySiegeCivilianCasualties(side *domain.BattleSide, militaryCasualties int64) int64 {
	if side.MilitiaCityID == nil || militaryCasualties <= 0 {
		return 0
	}
	cityID := *side.MilitiaCityID
	key := "civilians:" + cityID
	if state.civilianCasualtyCarry == nil {
		state.civilianCasualtyCarry = make(map[string]float64)
	}
	expected := float64(militaryCasualties)*constants.SiegeCivilianCasualtiesPerMilitaryLoss + state.civilianCasualtyCarry[key]
	requested := int64(math.Floor(expected))
	state.civilianCasualtyCarry[key] = expected - float64(requested)
	if requested <= 0 {
		return 0
	}
	res, err := state.Cluster.Request("city", cityID, messages.ApplyCivilianCasualtiesMessage{Count: requested})
	if err != nil {
		state.civilianCasualtyCarry[key] += float64(requested)
		slog.WarnContext(state.Ctx(), "failed to apply siege civilian casualties", "battle_id", state.Battle.BattleID, "city_id", cityID, "error", err)
		return 0
	}
	response, ok := res.(*messages.ApplyCivilianCasualtiesResponseMessage)
	if !ok {
		state.civilianCasualtyCarry[key] += float64(requested)
		return 0
	}
	return response.Applied
}

func (state *battleActor) recordCivilianCasualties(side *domain.BattleReportSide, count int64) {
	if side.Settlement != nil {
		side.Settlement.CivilianCasualties += count
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

func (state *battleActor) resolveIfFinished(ctx actor.Context, resolution domain.BattleReportResolution) bool {
	attackersActive := len(state.Battle.Attackers.ArmyIDs) > 0 || state.Battle.Attackers.MilitiaCount > 0
	defendersActive := len(state.Battle.Defenders.ArmyIDs) > 0 || state.Battle.Defenders.MilitiaCount > 0
	if attackersActive && defendersActive {
		return false
	}
	winners := state.Battle.Attackers.ArmyIDs
	attackersWon := attackersActive && !defendersActive
	draw := !attackersActive && !defendersActive
	if draw {
		resolution = domain.BattleReportResolutionMutualDestruction
	}
	if !attackersWon {
		winners = state.Battle.Defenders.ArmyIDs
	}
	state.createMailboxMessages(attackersWon, draw, resolution)
	state.endMilitiaBattle(state.Battle.Attackers.MilitiaCityID)
	state.endMilitiaBattle(state.Battle.Defenders.MilitiaCityID)
	for _, id := range winners {
		_ = state.Cluster.Tell("army", id, messages.LeaveBattleMessage{BattleID: state.Battle.BattleID})
	}
	state.stopRoundTimer()
	battles.Delete(state.Battle.BattleID)
	id := state.Battle.BattleID
	stream.Publish("", stream.StateUpdate{DeletedBattleID: &id})
	ctx.Stop(ctx.Self())
	return true
}

func (state *battleActor) createMailboxMessages(attackersWon, draw bool, resolution domain.BattleReportResolution) {
	endedAt := time.Now()
	state.refreshReportSettlement(&state.reportAttackers)
	state.refreshReportSettlement(&state.reportDefenders)
	recipients := appendUniqueStrings(state.reportAttackers.UserIDs, state.reportDefenders.UserIDs...)
	for _, userID := range recipients {
		role := domain.BattleReportRoleDefender
		onWinningSide := !attackersWon
		if contains(state.reportAttackers.UserIDs, userID) {
			role = domain.BattleReportRoleAttacker
			onWinningSide = attackersWon
		}
		outcome := domain.BattleReportOutcomeDefeat
		if draw {
			outcome = domain.BattleReportOutcomeDraw
		} else if onWinningSide {
			outcome = domain.BattleReportOutcomeVictory
		}
		engagement := domain.BattleReportEngagementField
		if state.reportAttackers.MilitiaCityID != nil || state.reportDefenders.MilitiaCityID != nil {
			engagement = domain.BattleReportEngagementSiege
		}
		report := &domain.BattleReport{
			BattleID:   state.Battle.BattleID,
			X:          state.Battle.X,
			Y:          state.Battle.Y,
			Role:       role,
			Outcome:    outcome,
			Engagement: engagement,
			Resolution: resolution,
			Attackers:  state.reportAttackers,
			Defenders:  state.reportDefenders,
			Rounds:     append([]domain.BattleReportRound(nil), state.reportRounds...),
			StartedAt:  state.Battle.StartedAt,
			EndedAt:    endedAt,
		}
		message := domain.MailboxMessage{
			MailboxMessageID: uuid.NewString(),
			RecipientID:      userID,
			CreatedAt:        endedAt,
			BattleReport:     report,
		}
		if err := state.Store.CreateMailboxMessage(state.Ctx(), message); err != nil {
			slog.ErrorContext(state.Ctx(), "failed to create battle mailbox message", "battle_id", state.Battle.BattleID, "user_id", userID, "error", err)
			continue
		}
		stream.Publish(userID, stream.StateUpdate{MailboxMessage: &message})
	}
}

func appendUniqueStrings(existing []string, values ...string) []string {
	result := append([]string(nil), existing...)
	for _, value := range values {
		if !contains(result, value) {
			result = append(result, value)
		}
	}
	return result
}

func (state *battleActor) endMilitiaBattle(cityID *string) {
	if cityID != nil {
		_ = state.Cluster.Tell("city", *cityID, messages.EndMilitiaBattleMessage{BattleID: state.Battle.BattleID})
	}
}

func (state *battleActor) publish() {
	state.syncLiveSummary()
	battles.Upsert(state.Battle)
	b := state.Battle
	stream.Publish("", stream.StateUpdate{Battle: &b})
}

func (state *battleActor) syncLiveSummary() {
	var attackerLast, defenderLast []domain.BattleReportLoss
	var attackerCivilians, defenderCivilians int64
	if len(state.reportRounds) > 0 {
		last := state.reportRounds[len(state.reportRounds)-1]
		attackerLast = last.AttackerLosses
		defenderLast = last.DefenderLosses
		attackerCivilians = last.AttackerCivilianCasualties
		defenderCivilians = last.DefenderCivilianCasualties
	}
	syncLiveBattleSide(&state.Battle.Attackers, state.reportAttackers, attackerLast, attackerCivilians)
	syncLiveBattleSide(&state.Battle.Defenders, state.reportDefenders, defenderLast, defenderCivilians)
	state.Battle.CompletedRounds = len(state.reportRounds)
}

func syncLiveBattleSide(side *domain.BattleSide, report domain.BattleReportSide, last []domain.BattleReportLoss, lastCivilians int64) {
	side.StartingTroops = make(map[domain.TroopType]int64)
	side.SurvivingTroops = make(map[domain.TroopType]int64)
	for _, army := range report.Armies {
		for troopType, count := range army.StartingTroops {
			side.StartingTroops[troopType] += count
		}
		for troopType, count := range army.SurvivingTroops {
			side.SurvivingTroops[troopType] += count
		}
	}
	side.StartingMilitia = report.StartingMilitia
	side.CumulativeLosses = domain.BattleLossSummary{
		Troops:    troopDifference(side.StartingTroops, side.SurvivingTroops),
		Militia:   max(report.StartingMilitia-report.SurvivingMilitia, 0),
		Civilians: reportCivilianCasualties(report),
	}
	side.LastRoundLosses = summarizeBattleLosses(last, lastCivilians)
}

func troopDifference(starting, surviving map[domain.TroopType]int64) map[domain.TroopType]int64 {
	losses := make(map[domain.TroopType]int64)
	for troopType, count := range starting {
		if lost := max(count-surviving[troopType], 0); lost > 0 {
			losses[troopType] = lost
		}
	}
	return losses
}

func summarizeBattleLosses(losses []domain.BattleReportLoss, civilians int64) domain.BattleLossSummary {
	summary := domain.BattleLossSummary{Troops: make(map[domain.TroopType]int64), Civilians: civilians}
	for _, loss := range losses {
		for troopType, count := range loss.Troops {
			summary.Troops[troopType] += count
		}
		summary.Militia += loss.Militia
	}
	return summary
}

func reportCivilianCasualties(report domain.BattleReportSide) int64 {
	if report.Settlement == nil {
		return 0
	}
	return report.Settlement.CivilianCasualties
}

func (state *battleActor) armNextRound(ctx actor.Context) {
	state.stopRoundTimer()
	state.Battle.NextTick = time.Now().Add(constants.BattleTickInterval)
	pid, system := ctx.Self(), ctx.ActorSystem()
	state.roundTimer = time.AfterFunc(constants.BattleTickInterval, func() {
		system.Root.Send(pid, messages.PeriodicOperationMessage{})
	})
}

func (state *battleActor) stopRoundTimer() {
	if state.roundTimer != nil {
		state.roundTimer.Stop()
		state.roundTimer = nil
	}
}
