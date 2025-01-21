package actors

import (
	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"math/rand"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

type CityActor struct {
	BaseActor
	City     models.City
	TilePIDs map[int]map[int]*actor.PID
	OwnerPID *actor.PID

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func (state *CityActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateCityMessage:
		state.City = msg.City
		state.TilePIDs = msg.TilePIDs
		state.OwnerPID = msg.OwnerPID

		if !msg.Restore {
			ctx.Send(state.database, messages.CreateCityMessage{
				City: state.City,
			})
		}
		ctx.Respond(messages.CreateCityResponseMessage{
			Error: nil,
		})
		state.startPeriodicOperation(ctx)

	case messages.UpdateOwnerPIDMessage:
		state.OwnerPID = msg.PID

	case messages.UpdateCityPopulationCapMessage:
		if state.City.Owner != "" {
			log.Println("Updating city population cap")
		}
		state.City.PopulationCap += float64(msg.Change)
		ctx.Respond(messages.UpdateCityPopulationCapResponseMessage{
			Error: nil,
		})

	case messages.GetCityMessage:
		ctx.Respond(messages.GetCityResponseMessage{
			City: state.City,
		})

	case messages.DeleteCityMessage:
		ctx.Send(state.database, messages.DeleteCityMessage{
			CityId: state.City.CityId,
		})
		ctx.Respond(messages.DeleteCityResponseMessage{
			Error: nil,
		})
		log.Printf("Shutting down CityActor for city: %s", state.City.Name)
		state.stopPeriodicOperation()
		ctx.Stop(ctx.Self())

	case messages.PeriodicOperationMessage:
		currentPopulation := float64(state.City.Population)
		populationCap := float64(state.City.PopulationCap)

		newPopulation := currentPopulation + (constants.POPULATION_GROWTH_RATE)*currentPopulation*(1-currentPopulation/populationCap)
		state.City.Population = newPopulation
		ctx.Send(state.database, &messages.UpdateCityMessage{
			City: state.City,
		})
	}
}

func (state *CityActor) startPeriodicOperation(ctx actor.Context) {
	go func() {
		// sleep for a random duration up to 10 seconds to attempt
		// creating an even distribution of database writing
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		time.Sleep(time.Duration(rnd.Intn(10)) * time.Second)

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
	close(state.stopTickerCh)
	state.ticker = nil
}
