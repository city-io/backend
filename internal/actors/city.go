package actors

import (
	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
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

		for _, row := range state.TilePIDs {
			for _, pid := range row {
				ctx.Send(pid, messages.UpdateTileCityPIDMessage{
					CityPID: ctx.Self(),
				})
			}
		}

		if !msg.Restore {
			err := state.createCity()
			ctx.Respond(messages.CreateCityResponseMessage{
				Error: err,
			})
		} else {
			ctx.Respond(messages.CreateCityResponseMessage{
				Error: nil,
			})
		}
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
		for _, row := range state.TilePIDs {
			for _, pid := range row {
				ctx.Send(pid, messages.UpdateTileCityPIDMessage{
					CityPID: nil,
				})
			}
		}
		result := state.db.Delete(&state.City)
		ctx.Respond(messages.DeleteCityResponseMessage{
			Error: result.Error,
		})
		log.Printf("Shutting down CityActor for city: %s", state.City.Name)
		state.stopPeriodicOperation()
		ctx.Stop(ctx.Self())

	case messages.PeriodicOperationMessage:
		currentPopulation := float64(state.City.Population)
		populationCap := float64(state.City.PopulationCap)

		newPopulation := currentPopulation + (constants.POPULATION_GROWTH_RATE)*currentPopulation*(1-currentPopulation/populationCap)
		state.City.Population = newPopulation
	}
}

func (state *CityActor) createCity() error {
	result := state.db.Create(&state.City)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (state *CityActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(3 * time.Second)

	go func() {
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
