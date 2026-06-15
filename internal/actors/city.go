package actors

import (
	"log/slog"
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
			buildingID := uuid.New().String()
			buildingType := domain.BuildingTypeCityCenter
			if msg.City.Type == domain.CityTypeTown {
				buildingType = domain.BuildingTypeTownCenter
			}
			buildingX := msg.City.StartX + msg.City.Size/2
			buildingY := msg.City.StartY + msg.City.Size/2
			building := domain.Building{
				BuildingID:        buildingID,
				CityID:            msg.City.CityID,
				Type:              string(buildingType),
				Level:             1,
				TargetLevel:       1,
				X:                 buildingX,
				Y:                 buildingY,
				ConstructionStart: domain.NullTime{Time: nil},
				ConstructionEnd:   domain.NullTime{Time: nil},
			}
			state.Cluster.Request("building", buildingID, &messages.CreateBuildingMessage{
				Building:  building,
				Restore:   false,
				Construct: false,
			})
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
		state.ws()

	case messages.CreditProductionMessage:
		if state.City.Owner == nil {
			ctx.Respond(messages.Ack{})
			return
		}
		if _, err := state.Cluster.Request("user", *state.City.Owner, messages.CreditUserMessage{
			Gold: msg.Gold,
			Food: msg.Food,
		}); err != nil {
			slog.ErrorContext(state.Ctx(), "failed to credit production to owner", "error", err)
			ctx.Respond(&messages.InternalError{})
			return
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
		currentPopulation := state.City.Population
		populationCap := state.City.PopulationCap
		if populationCap > 0 {
			newPopulation := currentPopulation + constants.PopulationGrowthRate*currentPopulation*(1-currentPopulation/populationCap)
			state.City.Population = newPopulation
		}
		state.Store.EnqueueCity(state.City)
		state.ws()
	}
}

func (state *cityActor) ws() {
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
		time.Sleep(time.Duration(rnd.Intn(constants.CityBackupFrequency)) * time.Second)

		state.ticker = time.NewTicker(constants.CityBackupFrequency * time.Second)
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
