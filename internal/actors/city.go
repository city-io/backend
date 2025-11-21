package actors

import (
	"math/rand"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
)

type CityActor struct {
	BaseActor
	City models.City

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func NewCityActor() ports.BaseActorInterface {
	return &CityActor{}
}

func (state *CityActor) ActorType() string {
	return "city"
}

func (state *CityActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case *messages.CreateCityMessage:
		state.City = msg.City

		if !msg.Restore {
			ctx.Send(state.Database, messages.CreateCityMessage{
				City: state.City,
			})
		}
		state.startPeriodicOperation(ctx)
		ctx.Respond(messages.CreateCityResponseMessage{})

	case messages.UpdateCityOwnerMessage:
		state.City.Owner = msg.Owner

	case messages.UpdateCityPopulationCapMessage:
		if state.City.Owner != "" {
			state.Log.Debug("Updating population cap", "city_id", state.City.CityID, "owner", state.City.Owner, "change", msg.Change)
		}
		state.City.PopulationCap += float64(msg.Change)

	case messages.GetCityMessage:
		ctx.Respond(&messages.GetCityResponseMessage{
			City: state.City,
		})

	case messages.DeleteCityMessage:
		ctx.Send(state.Database, &messages.DeleteCityMessage{
			CityID: state.City.CityID,
		})
		state.Log.Debug("Shutting down CityActor", "city_id", state.City.CityID)
		state.stopPeriodicOperation()
		ctx.Stop(ctx.Self())

	case messages.PeriodicOperationMessage:
		currentPopulation := float64(state.City.Population)
		populationCap := float64(state.City.PopulationCap)

		newPopulation := currentPopulation + (constants.POPULATION_GROWTH_RATE)*currentPopulation*(1-currentPopulation/populationCap)
		state.City.Population = newPopulation
		ctx.Send(state.Database, &messages.UpdateCityMessage{
			City: state.City,
		})
	}
}

func (state *CityActor) startPeriodicOperation(ctx actor.Context) {
	go func() {
		// sleep for a random duration up to 10 seconds to attempt
		// creating an even distribution of database writing
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		time.Sleep(time.Duration(rnd.Intn(constants.CITY_BACKUP_FREQUENCY)) * time.Second)

		state.ticker = time.NewTicker(constants.CITY_BACKUP_FREQUENCY * time.Second)
		state.stopTickerCh = make(chan struct{})

		for {
			select {
			case <-state.ticker.C:
				ctx.Send(ctx.Self(), messages.PeriodicOperationMessage{})
			case <-state.stopTickerCh:
				state.ticker.Stop()
				return
			}
		}
	}()
}

func (state *CityActor) stopPeriodicOperation() {
	select {
	case <-state.stopTickerCh:
	default:
		close(state.stopTickerCh)
	}
}
