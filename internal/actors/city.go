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
		state.populationContributions = make(map[string]float64)

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
		ctx.Respond(messages.Ack{})

	case messages.UpdateCityOwnerMessage:
		// The city is the sole authority for ownership; buildings and tiles no
		// longer cache it, so there is nothing to propagate.
		state.City.Owner = msg.Owner

	case messages.BuildingStateChangedMessage:
		if state.City.Owner != nil {
			b := msg.Building
			stream.Publish(*state.City.Owner, stream.StateUpdate{Building: &b})
			// Push the city alongside so derived stats (cap, food rates) update
			// without waiting for the next tick. The numbers are last-tick's
			// snapshot — they fully refresh on the next periodic tick.
			state.publish()
		}

	case messages.BuildingDestroyedMessage:
		delete(state.populationContributions, msg.BuildingID)
		var cap float64
		for _, p := range state.populationContributions {
			cap += p
		}
		state.City.PopulationCap = cap
		if state.City.Owner != nil {
			stream.Publish(*state.City.Owner, stream.StateUpdate{DeletedBuildingID: &msg.BuildingID})
			state.publish()
		}

	case messages.SetBuildingPopulationMessage:
		if state.populationContributions == nil {
			state.populationContributions = make(map[string]float64)
		}
		state.populationContributions[msg.BuildingID] = msg.Population
		var cap float64
		for _, p := range state.populationContributions {
			cap += p
		}
		state.City.PopulationCap = cap
		state.publish()

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

// spawnInitialBuilding kicks off a fully-built level-1 building inside the
// city block. Used during city creation for the center and (for capitals) the
// starter farm.
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
	upkeepPerHour := int64(math.Round(state.City.Population * float64(constants.FoodPerPopPerHour)))

	// Carry the sub-tick remainder so the actual per-tick demand averages
	// exactly to upkeepPerHour over time. See demandRemainder doc.
	scaled := upkeepPerHour*int64(tickSecs) + state.demandRemainder
	demand := scaled / int64(constants.SecondsPerHour)
	state.demandRemainder = scaled % int64(constants.SecondsPerHour)

	productionPerHour := production * int64(constants.SecondsPerHour) / int64(tickSecs)

	state.City.FoodProductionRate = productionPerHour
	state.City.FoodUpkeep = upkeepPerHour
	state.City.NetFoodFlow = productionPerHour - upkeepPerHour

	if production >= demand {
		// Local surplus: deposit, no starvation, scale growth by surplus.
		if surplus := production - demand; surplus > 0 {
			if err := state.Cluster.Tell("user", *state.City.Owner, messages.DepositFoodMessage{Amount: surplus}); err != nil {
				slog.ErrorContext(state.Ctx(), "failed to deposit surplus food to pool", "error", err)
			}
		}
		state.City.Starving = false
		var surplusRatio float64
		if demand > 0 {
			surplusRatio = float64(production-demand) / float64(demand)
		}
		state.growPopulation(false, 0, surplusRatio)
		return
	}

	// Local deficit: still draw from the pool (so the user's food drains as the
	// city imports), but the city is starving from its own perspective and its
	// population declines regardless of whether the pool covered the shortfall.
	shortfall := demand - production
	if _, err := state.Cluster.Request("user", *state.City.Owner, messages.RequestFoodFromPoolMessage{Amount: shortfall}); err != nil {
		slog.ErrorContext(state.Ctx(), "failed to request food from pool", "error", err)
	}
	state.City.Starving = true
	deficitRatio := float64(shortfall) / float64(demand)
	state.growPopulation(true, deficitRatio, 0)
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
		newPop = currentPopulation + constants.PopulationGrowthRate*currentPopulation*(1-currentPopulation/populationCap)*fedFactor
	}
	delta := newPop - currentPopulation
	state.City.PopulationGrowthRate = int64(math.Round(delta * float64(constants.SecondsPerHour) / float64(constants.CityTickInterval)))
	state.City.Population = newPop
}

// publish pushes the city's current state to the owning player's
// StreamState subscribers via the in-process pub/sub. Towns (no owner) skip
// the push. Call after any change the player should see without waiting for
// the next periodic tick — population cap shifts, building events, etc.
func (state *cityActor) publish() {
	if state.City.Owner == nil {
		return
	}
	c := state.City
	stream.Publish(*state.City.Owner, stream.StateUpdate{City: &c})
}

func (state *cityActor) startPeriodicOperation(ctx actor.Context) {
	pid := ctx.Self()
	system := ctx.ActorSystem()
	go func() {
		// sleep for a random duration up to 10 seconds to attempt
		// creating an even distribution of database writing
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		time.Sleep(time.Duration(rnd.Intn(constants.CityTickInterval)) * time.Second)

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
