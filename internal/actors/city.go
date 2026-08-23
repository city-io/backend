package actors

import (
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"

	"cityio/internal/constants"
	"cityio/internal/domain"
	"cityio/internal/messages"
	"cityio/internal/metrics"
	"cityio/internal/stream"
	"cityio/internal/utils"
)

type cityActor struct {
	baseActor
	City domain.City

	// populationContributions holds each building's absolute contribution to the
	// population cap, keyed by building ID. The cap is derived as their sum, so it
	// is idempotent under resends and fully rebuilt from buildings on restore.
	populationContributions map[string]float64

	// foodProduction holds each building's current per-hour food capacity.
	// Structural food state uses this stable rate; pendingFoodIncome remains the
	// integer accounting channel for actual resource transfers.
	foodProduction map[string]int64

	// armyUpkeep holds each army's food upkeep (per hour) currently attributed
	// to this city, keyed by army ID. The city's army food demand is their sum,
	// so it is idempotent under resends. Armies re-attribute themselves to the
	// nearest owned city as they move.
	armyUpkeep map[string]int64

	// pendingFoodIncome holds food produced by this city's buildings since the
	// last tick. It is consumed locally first; only the surplus is deposited to
	// the user's pool.
	pendingFoodIncome int64

	// demandRemainder carries the sub-tick fractional part of the per-hour
	// upkeep into the next tick. Per-hour upkeep × tickSeconds rarely divides
	// cleanly into SecondsPerHour (because population is fractional), so
	// without this remainder the truncated per-tick demand silently discards
	// 0–1 food per tick — the pool would never drain at the displayed
	// per-hour deficit rate. Carrying the remainder makes the long-run pool
	// drain exactly match the displayed FoodUpkeep.
	demandRemainder int64
	taxRemainder    int64

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func NewCityActor() BaseActorInterface {
	return &cityActor{}
}

func (state *cityActor) ActorType() string {
	return "city"
}

func (state *cityActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case *messages.CreateCityMessage:
		state.City = msg.City
		state.City.TaxIncomeRate = constants.TaxIncomePerHour(state.City)
		state.populationContributions = make(map[string]float64)
		state.foodProduction = make(map[string]int64)
		state.armyUpkeep = make(map[string]int64)

		if !msg.Restore {
			if err := state.Store.CreateCity(state.Ctx(), msg.City); err != nil {
				slog.ErrorContext(state.Ctx(), "failed to persist city create", "city_id", msg.City.CityID, "error", err)
			}
			centerType := domain.BuildingTypeCityCenter
			if msg.City.Type == domain.CityTypeTown {
				centerType = domain.BuildingTypeTownCenter
			}
			centerX := msg.City.StartX + msg.City.Size/2
			centerY := msg.City.StartY + msg.City.Size/2
			state.spawnInitialBuilding(centerType, centerX, centerY)

			// Player capitals ship with one farm so they're self-sustaining at the
			// initial population: pop=250 demands ~33 food/tick, one L1 farm
			// produces ~33 food/tick. Towns don't need one (they're unowned).
			if msg.City.Type == domain.CityTypeCity {
				state.spawnInitialBuilding(domain.BuildingTypeFarm, msg.City.StartX+1, msg.City.StartY+1)
				// TODO: Temporary development bootstrap; remove the free barracks once
				// the early-game city progression is finalized.
				state.spawnInitialBuilding(domain.BuildingTypeBarracks, msg.City.StartX+1, msg.City.StartY+2)
			}
		}
		state.startPeriodicOperation(ctx)

		startX := msg.City.StartX
		startY := msg.City.StartY
		size := msg.City.Size
		for dx := range size {
			for dy := range size {
				idx := utils.GetTileIndex(startX+dx, startY+dy)

				_, err := state.Cluster.Request("tile", idx, messages.UpdateTileCityMessage{
					CityID: msg.City.CityID,
				})
				if err != nil {
					slog.ErrorContext(state.Ctx(), "failed to signal tile of city presence", "city_id", msg.City.CityID, "tile", idx, "error", err)
				}
			}
		}
		if state.City.Owner != nil {
			state.recordExploration(*state.City.Owner, domain.Vision{Cities: []domain.City{state.City}})
		}
		ctx.Respond(messages.Ack{})

	case messages.UpdateCityOwnerMessage:
		// The city is the sole authority for ownership; buildings and tiles no
		// longer cache it, so there is nothing to propagate.
		state.City.Owner = msg.Owner
		if state.City.Owner != nil {
			state.recordExploration(*state.City.Owner, domain.Vision{Cities: []domain.City{state.City}})
		}
		state.Store.EnqueueCity(state.City)
		state.publishWorld()

	case messages.CaptureCityMessage:
		owner := msg.Owner
		state.City.Owner = &owner
		state.recordExploration(owner, domain.Vision{Cities: []domain.City{state.City}})
		state.Store.EnqueueCity(state.City)
		state.publishWorld()
		ctx.Respond(messages.Ack{})

	case messages.UpdateCityPolicyMessage:
		if state.City.GarrisonBattleID != nil {
			ctx.Respond(&messages.CityPolicyLockedError{})
			return
		}
		if msg.GarrisonPercent < 0 || msg.GarrisonPercent > constants.MaxGarrisonPercent || msg.TaxRatePercent < 0 || msg.TaxRatePercent > constants.MaxTaxRatePercent {
			ctx.Respond(&messages.InvalidCityPolicyError{})
			return
		}
		garrisonChanged := state.City.GarrisonPercent != msg.GarrisonPercent
		state.City.GarrisonPercent = msg.GarrisonPercent
		state.City.TaxRatePercent = msg.TaxRatePercent
		// Raising the target reserves future growth for the garrison; it never
		// conjures defenders from civilians. Lowering it releases any excess.
		if garrisonChanged {
			target := constants.GarrisonTarget(state.City)
			state.City.GarrisonPopulation = min(state.City.GarrisonPopulation, target)
		}
		state.City.TaxIncomeRate = constants.TaxIncomePerHour(state.City)
		state.Store.EnqueueCity(state.City)
		state.publish()
		ctx.Respond(&messages.GetCityResponseMessage{City: state.City})

	case messages.ApplyGarrisonCasualtiesMessage:
		survived := state.applyGarrisonCasualties(msg.Count)
		state.Store.EnqueueCity(state.City)
		state.publishWorld()
		ctx.Respond(&messages.ApplyGarrisonCasualtiesResponseMessage{Survived: survived})

	case messages.BeginGarrisonBattleMessage:
		if state.City.GarrisonBattleID != nil {
			ctx.Respond(&messages.BeginGarrisonBattleResponseMessage{BattleID: *state.City.GarrisonBattleID, Count: int64(math.Floor(state.City.GarrisonPopulation))})
			return
		}
		if state.City.GarrisonPopulation < 1 {
			ctx.Respond(&messages.BeginGarrisonBattleResponseMessage{})
			return
		}
		battleID := msg.BattleID
		state.City.GarrisonBattleID = &battleID
		ctx.Respond(&messages.BeginGarrisonBattleResponseMessage{BattleID: battleID, Count: int64(math.Floor(state.City.GarrisonPopulation))})

	case messages.EndGarrisonBattleMessage:
		if state.City.GarrisonBattleID != nil && *state.City.GarrisonBattleID == msg.BattleID {
			state.City.GarrisonBattleID = nil
		}

	case messages.BuildingStateChangedMessage:
		// Real state change (created, upgrade started, upgrade complete) — push
		// the building proto and the city snapshot together so the player sees
		// both the new level and the cap/food-rate fields it implies in the
		// same emit.
		if state.City.Owner != nil {
			b := msg.Building
			stream.Publish(*state.City.Owner, stream.StateUpdate{Building: &b})
			state.publish()
		}

	case messages.BuildingDestroyedMessage:
		delete(state.populationContributions, msg.BuildingID)
		delete(state.foodProduction, msg.BuildingID)
		var cap float64
		for _, p := range state.populationContributions {
			cap += p
		}
		state.City.PopulationCap = cap
		state.City.FoodProductionRate = state.foodProductionTotal()
		state.City.NetFoodFlow = state.City.FoodProductionRate - state.City.FoodUpkeep
		// Real state change — emit the deletion id and the updated city
		// snapshot (with the lower cap) together.
		if state.City.Owner != nil {
			stream.Publish(*state.City.Owner, stream.StateUpdate{DeletedBuildingID: &msg.BuildingID})
			state.publish()
		}

	case messages.SetBuildingPopulationMessage:
		// Buildings re-report the same contribution on every periodic tick;
		// only publish when the value actually changes (initial register,
		// upgrade complete, building destroyed). Otherwise we'd fan out N
		// publishes per 3s for an N-building city with nothing visible to
		// say. cityCenter / townCenter / house all use this path.
		if state.populationContributions == nil {
			state.populationContributions = make(map[string]float64)
		}
		if existing, ok := state.populationContributions[msg.BuildingID]; ok && existing == msg.Population {
			return
		}
		state.populationContributions[msg.BuildingID] = msg.Population
		var cap float64
		for _, p := range state.populationContributions {
			cap += p
		}
		state.City.PopulationCap = cap
		state.publish()

	case messages.SetBuildingFoodProductionMessage:
		if state.foodProduction == nil {
			state.foodProduction = make(map[string]int64)
		}
		existing, exists := state.foodProduction[msg.BuildingID]
		if existing == msg.AmountPerHour && (exists || msg.AmountPerHour == 0) {
			return
		}
		if msg.AmountPerHour == 0 {
			delete(state.foodProduction, msg.BuildingID)
		} else {
			state.foodProduction[msg.BuildingID] = msg.AmountPerHour
		}
		state.City.FoodProductionRate = state.foodProductionTotal()
		state.City.NetFoodFlow = state.City.FoodProductionRate - state.City.FoodUpkeep
		state.publish()

	case messages.RecruitPopulationMessage:
		if err := state.recruitPopulation(msg.Count); err != nil {
			ctx.Respond(err)
			return
		}
		state.Store.EnqueueCity(state.City)
		state.publish()
		ctx.Respond(messages.Ack{})

	case messages.ReturnRecruitsMessage:
		state.City.Population += float64(msg.Count)
		state.City.TaxIncomeRate = constants.TaxIncomePerHour(state.City)
		state.Store.EnqueueCity(state.City)
		state.publish()
		if ctx.Sender() != nil {
			ctx.Respond(messages.Ack{})
		}

	case messages.SetArmyUpkeepMessage:
		if state.armyUpkeep == nil {
			state.armyUpkeep = make(map[string]int64)
		}
		if existing, ok := state.armyUpkeep[msg.ArmyID]; ok && existing == msg.Amount {
			return
		}
		state.armyUpkeep[msg.ArmyID] = msg.Amount

	case messages.RemoveArmyUpkeepMessage:
		delete(state.armyUpkeep, msg.ArmyID)

	case messages.CreditProductionMessage:
		if state.City.Owner == nil {
			ctx.Respond(messages.Ack{})
			return
		}
		state.pendingFoodIncome += msg.Food
		if msg.Gold > 0 {
			if _, err := state.Cluster.Request("user", *state.City.Owner, messages.CreditUserMessage{
				Gold: msg.Gold,
			}); err != nil {
				slog.ErrorContext(state.Ctx(), "failed to credit gold production to owner", "error", err)
				ctx.Respond(&messages.InternalError{})
				return
			}
		}
		ctx.Respond(messages.Ack{})

	case messages.DeductOwnerGoldMessage:
		if state.City.Owner == nil {
			ctx.Respond(&messages.InternalError{})
			return
		}
		res, err := state.Cluster.Request("user", *state.City.Owner, messages.CheckAndDeductGoldMessage{
			Amount: msg.Amount,
		})
		if err != nil {
			slog.ErrorContext(state.Ctx(), "failed to deduct gold from owner", "error", err)
			ctx.Respond(&messages.InternalError{})
			return
		}
		ctx.Respond(res)

	case messages.CreditOwnerGoldMessage:
		if state.City.Owner == nil {
			ctx.Respond(&messages.InternalError{})
			return
		}
		res, err := state.Cluster.Request("user", *state.City.Owner, messages.RefundUserGoldMessage{Amount: msg.Amount})
		if err != nil {
			slog.ErrorContext(state.Ctx(), "failed to refund gold to owner", "error", err)
			ctx.Respond(&messages.InternalError{})
			return
		}
		ctx.Respond(res)

	case messages.ReconcileTilesMessage:
		for dx := range state.City.Size {
			for dy := range state.City.Size {
				idx := utils.GetTileIndex(state.City.StartX+dx, state.City.StartY+dy)
				if err := state.Cluster.Tell("tile", idx, messages.UpdateTileCityMessage{CityID: state.City.CityID}); err != nil {
					slog.ErrorContext(state.Ctx(), "failed to reconcile tile city index", "tile", idx, "error", err)
				}
			}
		}

	case messages.GetCityMessage:
		ctx.Respond(&messages.GetCityResponseMessage{
			City: state.City,
		})

	case messages.DeleteCityMessage:
		// TODO: should a city be able to be fully removed?
		// ctx.Send(state.Cluster.DB(), messages.DeleteCityMessage{
		// CityID: state.City.CityID,
		// })
		slog.DebugContext(state.Ctx(), "shutting down CityActor", "city_id", state.City.CityID)
		state.stopPeriodicOperation()
		ctx.Stop(ctx.Self())

	case messages.PeriodicOperationMessage:
		state.tickFoodAndPopulation()
		state.Store.EnqueueCity(state.City)
		state.publish()
	}
}

func (state *cityActor) recruitPopulation(count int64) *messages.InsufficientPopulationError {
	available := constants.RecruitablePopulation(state.City)
	if count > available {
		return &messages.InsufficientPopulationError{Available: available, Requested: count}
	}
	state.City.Population -= float64(count)
	state.City.TaxIncomeRate = constants.TaxIncomePerHour(state.City)
	return nil
}

func (state *cityActor) applyGarrisonCasualties(count int64) bool {
	removed := min(float64(count), state.City.GarrisonPopulation)
	state.City.GarrisonPopulation -= removed
	state.City.Population = max(state.City.Population-removed, 0)
	state.City.TaxIncomeRate = constants.TaxIncomePerHour(state.City)
	return state.City.GarrisonPopulation >= 1
}

// armyUpkeepTotal is the sum of food upkeep (per hour) for all armies currently
// attributed to this city.
func (state *cityActor) armyUpkeepTotal() int64 {
	var sum int64
	for _, u := range state.armyUpkeep {
		sum += u
	}
	return sum
}

func (state *cityActor) foodProductionTotal() int64 {
	var sum int64
	for _, production := range state.foodProduction {
		sum += production
	}
	return sum
}

// spawnInitialBuilding kicks off a fully-built level-1 building inside the
// city block. Used during city creation for its fully-built starter buildings.
func (state *cityActor) spawnInitialBuilding(buildingType domain.BuildingType, x, y int) {
	id := uuid.New().String()
	state.Cluster.Request("building", id, &messages.CreateBuildingMessage{
		Building: domain.Building{
			BuildingID:        id,
			CityID:            state.City.CityID,
			Type:              string(buildingType),
			Level:             1,
			TargetLevel:       1,
			X:                 x,
			Y:                 y,
			ConstructionStart: domain.NullTime{Time: nil},
			ConstructionEnd:   domain.NullTime{Time: nil},
		},
		Restore:   false,
		Construct: false,
	})
}

// tickFoodAndPopulation runs the per-tick food loop for the city: consume the
// city's own production first, deposit any surplus to the user pool or request
// the shortfall from it, then grow or decline the population.
//
// Growth/decline is decided by *local* production vs demand — the pool can no
// longer rescue a deficit city's population. A city that imports its food
// holds the pool drain but doesn't grow; if production covers demand the
// surplus accelerates growth proportionally up to SurplusGrowthBonus.
func (state *cityActor) tickFoodAndPopulation() {
	start := time.Now()
	defer func() {
		metrics.CityTickDurationSeconds.Observe(time.Since(start).Seconds())
	}()

	if state.City.Owner == nil {
		state.City.FoodProductionRate = 0
		state.City.FoodUpkeep = 0
		state.City.NetFoodFlow = 0
		state.City.Starving = false
		state.growPopulation(false, 0, 0)
		return
	}

	production := state.pendingFoodIncome
	state.pendingFoodIncome = 0

	tickSecs := constants.CityTickInterval
	// Everyone physically resident in the city—including its garrison—eats
	// here. Field armies are charged separately to their nearest settlement.
	upkeepPerHour := int64(math.Round(state.City.Population*float64(constants.FoodPerPopPerHour))) + state.armyUpkeepTotal()

	// Carry the sub-tick remainder so actual inventory transfers average to the
	// precise hourly upkeep. The remainder does not determine biological state.
	demand := rateAmountForTick(upkeepPerHour, tickSecs, &state.demandRemainder)
	productionPerHour := state.foodProductionTotal()

	state.City.FoodProductionRate = productionPerHour
	state.City.FoodUpkeep = upkeepPerHour
	state.City.NetFoodFlow = productionPerHour - upkeepPerHour
	state.City.TaxIncomeRate = constants.TaxIncomePerHour(state.City)
	if tax := rateAmountForTick(state.City.TaxIncomeRate, tickSecs, &state.taxRemainder); tax > 0 {
		if _, err := state.Cluster.Request("user", *state.City.Owner, messages.CreditUserMessage{Gold: tax}); err != nil {
			slog.ErrorContext(state.Ctx(), "failed to credit city tax income", "city_id", state.City.CityID, "error", err)
		}
	}

	// Settle whole food units independently from the stable rate comparison.
	if surplus := production - demand; surplus > 0 {
		if err := state.Cluster.Tell("user", *state.City.Owner, messages.DepositFoodMessage{Amount: surplus}); err != nil {
			slog.ErrorContext(state.Ctx(), "failed to deposit surplus food to pool", "error", err)
		} else {
			metrics.FoodDepositedTotal.Add(float64(surplus))
		}
	} else if shortfall := demand - production; shortfall > 0 {
		res, err := state.Cluster.Request("user", *state.City.Owner, messages.RequestFoodFromPoolMessage{Amount: shortfall})
		if err != nil {
			slog.ErrorContext(state.Ctx(), "failed to request food from pool", "error", err)
			metrics.FoodPoolGrantsTotal.WithLabelValues("empty").Inc()
		} else if resp, ok := res.(messages.RequestFoodFromPoolResponse); ok {
			metrics.FoodWithdrawnTotal.Add(float64(resp.Granted))
			switch {
			case resp.Granted >= shortfall:
				metrics.FoodPoolGrantsTotal.WithLabelValues("full").Inc()
			case resp.Granted > 0:
				metrics.FoodPoolGrantsTotal.WithLabelValues("partial").Inc()
			default:
				metrics.FoodPoolGrantsTotal.WithLabelValues("empty").Inc()
			}
		}
	}

	balance := calculateCityFoodBalance(productionPerHour, upkeepPerHour)
	state.City.Starving = balance.starving
	state.growPopulation(balance.starving, balance.deficitRatio, balance.surplusRatio)
}

type cityFoodBalance struct {
	starving     bool
	deficitRatio float64
	surplusRatio float64
}

func calculateCityFoodBalance(productionPerHour, upkeepPerHour int64) cityFoodBalance {
	if productionPerHour < upkeepPerHour {
		return cityFoodBalance{
			starving:     true,
			deficitRatio: float64(upkeepPerHour-productionPerHour) / float64(upkeepPerHour),
		}
	}
	if upkeepPerHour > 0 {
		return cityFoodBalance{surplusRatio: float64(productionPerHour-upkeepPerHour) / float64(upkeepPerHour)}
	}
	return cityFoodBalance{}
}

func rateAmountForTick(perHour int64, tickSeconds int, remainder *int64) int64 {
	scaled := perHour*int64(tickSeconds) + *remainder
	*remainder = scaled % int64(constants.SecondsPerHour)
	return scaled / int64(constants.SecondsPerHour)
}

// growPopulation moves the population for one tick: logistic growth scaled by
// a food-surplus bonus when fed, or a decline scaled by the local deficit
// ratio when not. Records the per-tick delta as a per-hour rate on the city
// so clients can render the trend without reverse-engineering the formulas.
func (state *cityActor) growPopulation(starving bool, deficitRatio, surplusRatio float64) {
	currentPopulation := state.City.Population
	populationCap := state.City.PopulationCap
	if populationCap <= 0 {
		state.City.PopulationGrowthRate = 0
		return
	}
	var newPop float64
	if starving {
		newPop = currentPopulation * (1 - constants.StarvationDeclineRate*deficitRatio)
	} else {
		// Surplus bonus saturates at 100% extra production (surplusRatio = 1.0).
		// fedFactor goes from 1.0 (just covered) up to 1 + SurplusGrowthBonus
		// at saturation; beyond that, more farms give no further speedup.
		bonus := math.Min(surplusRatio, 1.0) * constants.SurplusGrowthBonus
		fedFactor := 1.0 + bonus
		newPop = currentPopulation + constants.PopulationGrowthRate*currentPopulation*(1-currentPopulation/populationCap)*fedFactor*constants.PositiveGrowthMultiplier(state.City.TaxRatePercent)
	}
	delta := newPop - currentPopulation
	if delta > 0 && state.City.GarrisonBattleID == nil {
		shortfall := max(constants.GarrisonTarget(state.City)-state.City.GarrisonPopulation, 0)
		state.City.GarrisonPopulation += min(delta, shortfall)
	} else if state.City.GarrisonPopulation > newPop {
		state.City.GarrisonPopulation = max(newPop, 0)
	}
	state.City.PopulationGrowthRate = int64(math.Round(delta * float64(constants.SecondsPerHour) / float64(constants.CityTickInterval)))
	state.City.Population = newPop
}

// publish pushes the city's current state to the owning player's
// StreamState subscribers via the in-process pub/sub. Towns (no owner) skip
// the push. Called on real state changes (building created/upgraded/destroyed,
// per-tick food/growth recompute) — not on every periodic message a building
// fires at us, so we don't fan out a publish per building per tick.
func (state *cityActor) publish() {
	if state.City.Owner == nil {
		return
	}
	c := state.City
	stream.Publish(*state.City.Owner, stream.StateUpdate{City: &c})
}

func (state *cityActor) publishWorld() {
	c := state.City
	owner := ""
	if state.City.Owner != nil {
		owner = *state.City.Owner
	}
	stream.Publish(owner, stream.StateUpdate{City: &c})
}

func (state *cityActor) startPeriodicOperation(ctx actor.Context) {
	pid := ctx.Self()
	system := ctx.ActorSystem()
	go func() {
		// Sleep [1, CityTickInterval-1] seconds before the first tick so the
		// city is always cleanly offset from the (parallel) building tickers
		// that fire on the same 3s cadence. Without this offset, a city tick
		// can fire at the same moment as a farm tick and the farm's
		// CreditProductionMessage races the city's PeriodicOperationMessage —
		// when the city's tick wins, production=0 and the city phantom-starves.
		// Also doubles as jitter for DB-flush load distribution.
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		minOffset := 1
		jitter := rnd.Intn(constants.CityTickInterval - minOffset)
		time.Sleep(time.Duration(minOffset+jitter) * time.Second)

		state.ticker = time.NewTicker(constants.CityTickInterval * time.Second)
		state.stopTickerCh = make(chan struct{})

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

func (state *cityActor) stopPeriodicOperation() {
	select {
	case <-state.stopTickerCh:
	default:
		close(state.stopTickerCh)
	}
}
