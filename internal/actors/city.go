package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
	"gorm.io/gorm"
)

type CityActor struct {
	Db       *gorm.DB
	City     models.City
	TilePIDs map[int]map[int]*actor.PID
}

func NewCityActor(db *gorm.DB) *CityActor {
	actor := &CityActor{
		Db: db,
	}
	return actor
}

func (state *CityActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateCityMessage:
		state.City = msg.City
		state.TilePIDs = msg.TilePIDs

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
		}

	case messages.GetCityMessage:
		ctx.Respond(messages.GetCityResponseMessage{
			City: state.City,
		})
	}
}

func (state *CityActor) createCity() error {
	result := state.Db.Create(&state.City)
	if result.Error != nil {
		log.Printf("Error creating city: %s", result.Error)
		return result.Error
	}
	return nil
}
