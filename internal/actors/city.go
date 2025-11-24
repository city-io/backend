package actors

import (
	"math/rand"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
	"cityio/internal/utils"
)

type cityActor struct {
	BaseActor
	City models.City

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func NewCityActor() ports.BaseActorInterface {
	return &cityActor{}
}

func (state *cityActor) ActorType() string {
	return "city"
}

func (state *cityActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case *messages.CreateCityMessage:
		state.City = msg.City

		if !msg.Restore {
			ctx.Send(state.Cluster.DB(), msg)
		}
		state.startPeriodicOperation(ctx)
		ctx.Respond(messages.Ack{})

		startX := msg.City.StartX
		startY := msg.City.StartY
		size := msg.City.Size
		for x := startX; x <= startX+size; x++ {
			for y := startY; y <= startY+size; y++ {
				idx := utils.GetTileIndex(x, y)

				_, err := state.Cluster.Request("tile", idx, messages.UpdateTileCityMessage{
					CityID: msg.City.CityID,
				})
				if err != nil {
					state.Log.Error("failed to signal tile of city presence", "city_id", msg.City.CityID, "tile", idx, "error", err)
				}
			}
		}

	case messages.UpdateCityOwnerMessage:
		state.City.Owner = &msg.Owner

	case messages.UpdateCityPopulationCapMessage:
		if state.City.Owner != nil {
			state.Log.Debug("updating population cap", "city_id", state.City.CityID, "owner", state.City.Owner, "change", msg.Change)
		}
		state.City.PopulationCap += float64(msg.Change)

	case messages.GetCityMessage:
		ctx.Respond(&messages.GetCityResponseMessage{
			City: state.City,
		})

	case messages.DeleteCityMessage:
		ctx.Send(state.Cluster.DB(), &messages.DeleteCityMessage{
			CityID: state.City.CityID,
		})
		state.Log.Debug("shutting down CityActor", "city_id", state.City.CityID)
		state.stopPeriodicOperation()
		ctx.Stop(ctx.Self())

	case messages.PeriodicOperationMessage:
		currentPopulation := float64(state.City.Population)
		populationCap := float64(state.City.PopulationCap)

		newPopulation := currentPopulation + (constants.PopulationGrowthRate)*currentPopulation*(1-currentPopulation/populationCap)
		state.City.Population = newPopulation
		ctx.Send(state.Cluster.DB(), &messages.UpdateCityMessage{
			City: state.City,
		})
	}
}

func (state *cityActor) startPeriodicOperation(ctx actor.Context) {
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
				ctx.Send(ctx.Self(), messages.PeriodicOperationMessage{})
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
