package actors

import (
	"context"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/ports"
)

type DatabaseActor struct {
	BaseActor
	db database.Querier

	userBuffer []models.User
	cityBuffer []models.City
	// armyBuffer []models.Army

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func NewDatabaseActor(db database.Querier) ports.BaseActorInterface {
	return &DatabaseActor{
		db:         db,
		userBuffer: make([]models.User, 0),
		cityBuffer: make([]models.City, 0),
		// armyBuffer:   make([]models.Army, 0),
		stopTickerCh: make(chan struct{}),
	}
}

func (state *DatabaseActor) ActorType() string {
	return "database"
}

func (state *DatabaseActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.InitDatabaseMessage:
		state.startPeriodicOperation(ctx)

	case *messages.RegisterUserMessage:
		err := state.db.CreateUser(context.Background(), database.CreateUserParams{
			UserID:   msg.User.UserID,
			Email:    msg.User.Email,
			Username: msg.User.Username,
			Password: msg.User.Password,
		})
		if err != nil {
			state.Log.Error("Error creating user in db", "error", err)
		}
	case *messages.UpdateUserMessage:
		err := state.db.UpdateUser(context.Background(), database.UpdateUserParams{
			UserID:   msg.User.UserID,
			Username: msg.User.Username,
			Gold:     msg.User.Gold,
			Food:     msg.User.Food,
		})
		if err != nil {
			state.Log.Error("Error updating user in db", "error", err)
		}
	case messages.DeleteUserMessage:
		err := state.db.DeleteUser(context.Background(), msg.UserID)
		if err != nil {
			state.Log.Error("Error deleting user in db", "error", err)
		}

	case messages.CreateCityMessage:
		err := state.db.CreateCity(context.Background(), database.CreateCityParams{
			CityID:        msg.City.CityID,
			Type:          msg.City.Type,
			Owner:         msg.City.Owner,
			Name:          msg.City.Name,
			Population:    msg.City.Population,
			PopulationCap: msg.City.PopulationCap,
			Coordinates:   msg.City.StartX,
			Coordinates_2: msg.City.StartY,
			Size:          int32(msg.City.Size),
		})
		if err != nil {
			state.Log.Error("Error creating city in db", "error", err)
		}
	case messages.DeleteCityMessage:
		err := state.db.DeleteCity(context.Background(), msg.CityID)
		if err != nil {
			state.Log.Error("Error deleting city in db", "error", err)
		}
	case *messages.UpdateCityMessage:
		state.cityBuffer = append(state.cityBuffer, msg.City)

	case messages.PeriodicOperationMessage:
		cityBatchSize := 5000
		if len(state.cityBuffer) > 0 {
			for i := 0; i < len(state.cityBuffer); i += cityBatchSize {
				// end := min(i+cityBatchSize, len(state.cityBuffer))
				// err := state.db.BatchUpdateCities(context.Background(), database.BatchUpdateCitiesParams{})
				// if result := state.db.Save(state.cityBuffer[i:end]); result.Error != nil {
				// log.Printf("Error creating cities: %s", result.Error)
				// }
			}
			state.cityBuffer = make([]models.City, 0)
		}
	}
}

func (state *DatabaseActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.DBBackupFrequency * time.Second)

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
