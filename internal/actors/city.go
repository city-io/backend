package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
	"gorm.io/gorm"
)

type CityActor struct {
	Db   *gorm.DB
	City models.City
}

func NewCityActor(db *gorm.DB) *CityActor {
	actor := &CityActor{
		City: models.City{},
		Db:   db,
	}
	return actor
}

func (state *CityActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateCityMessage:
		state.City = msg.City
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
