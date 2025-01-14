package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
)

type CityActor struct {
	BaseActor
	City     models.City
	TilePIDs map[int]map[int]*actor.PID
	OwnerPID *actor.PID
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

	case messages.UpdateOwnerPIDMessage:
		state.OwnerPID = msg.PID

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
		ctx.Stop(ctx.Self())
	}
}

func (state *CityActor) createCity() error {
	result := state.db.Create(&state.City)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
