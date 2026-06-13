package actors

import (
	"log/slog"
	"time"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/constants"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"
)

type databaseActor struct {
	baseActor
	db database.Querier

	// use map to only preserve latest update
	userBuffer map[string]models.User
	cityBuffer map[string]models.City

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func NewDatabaseActor(db database.Querier) BaseActorInterface {
	return &databaseActor{
		db:           db,
		userBuffer:   make(map[string]models.User),
		cityBuffer:   make(map[string]models.City),
		stopTickerCh: make(chan struct{}),
	}
}

func (state *databaseActor) ActorType() string {
	return "database"
}

func (state *databaseActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.InitDatabaseMessage:
		state.startPeriodicOperation(ctx)

	case *messages.CreateUserMessage:
		err := state.db.CreateUser(state.Ctx(), database.CreateUserParams{
			UserID:   msg.User.UserID,
			Email:    msg.User.Email,
			Username: msg.User.Username,
			Password: msg.User.Password,
		})
		if err != nil {
			slog.ErrorContext(state.Ctx(), "error creating user in db", "error", err)
		}
	case *messages.UpdateUserMessage:
		state.userBuffer[msg.User.UserID] = msg.User
	case messages.DeleteUserMessage:
		err := state.db.DeleteUser(state.Ctx(), msg.UserID)
		if err != nil {
			slog.ErrorContext(state.Ctx(), "error deleting user in db", "error", err)
		}

	case *messages.CreateCityMessage:
		err := state.db.CreateCity(state.Ctx(), database.CreateCityParams{
			CityID:        msg.City.CityID,
			Type:          string(msg.City.Type),
			Owner:         msg.City.Owner,
			Name:          msg.City.Name,
			Population:    msg.City.Population,
			PopulationCap: msg.City.PopulationCap,
			StartX:        int32(msg.City.StartX),
			StartY:        int32(msg.City.StartY),
			Size:          int32(msg.City.Size),
		})
		if err != nil {
			slog.ErrorContext(state.Ctx(), "error creating city in db", "error", err)
		}
	case messages.DeleteCityMessage:
		err := state.db.DeleteCity(state.Ctx(), msg.CityID)
		if err != nil {
			slog.ErrorContext(state.Ctx(), "error deleting city in db", "error", err)
		}
	case *messages.UpdateCityMessage:
		state.cityBuffer[msg.City.CityID] = msg.City

	case messages.GetEmptyCityBlockMessage:
		row, err := state.db.FindEmptyCityBlock(state.Ctx(), database.FindEmptyCityBlockParams{
			MapWidth:  constants.MapSize,
			MapHeight: constants.MapSize,
			Size:      int32(msg.Size),
		})
		if err != nil {
			slog.ErrorContext(state.Ctx(), "error fetching empty city block from db", "error", err)
			return
		}
		ctx.Respond(messages.GetEmptyCityBlockResponseMessage{
			X: int(row.X),
			Y: int(row.Y),
		})

	case *messages.CreateBuildingMessage:
		err := state.db.CreateBuilding(state.Ctx(), database.CreateBuildingParams{
			BuildingID:        msg.Building.BuildingID,
			CityID:            msg.Building.CityID,
			Type:              msg.Building.Type,
			Level:             int32(msg.Building.Level),
			TargetLevel:       int32(msg.Building.TargetLevel),
			X:                 int32(msg.Building.X),
			Y:                 int32(msg.Building.Y),
			ConstructionStart: msg.Building.ConstructionStart.ToPG(),
			ConstructionEnd:   msg.Building.ConstructionEnd.ToPG(),
		})
		if err != nil {
			slog.ErrorContext(state.Ctx(), "error creating building in db", "error", err)
		}

	case messages.PeriodicOperationMessage:
		cityBatchSize := 5000
		cities := make([]models.City, 0, len(state.cityBuffer))
		for _, c := range state.cityBuffer {
			cities = append(cities, c)
		}
		for i := 0; i < len(cities); i += cityBatchSize {
			end := min(i+cityBatchSize, len(cities))
			chunk := cities[i:end]

			params := database.BatchUpdateCitiesParams{
				CityIds:        make([]string, 0, len(chunk)),
				Types:          make([]string, 0, len(chunk)),
				Owners:         make([]string, 0, len(chunk)),
				Names:          make([]string, 0, len(chunk)),
				Populations:    make([]float64, 0, len(chunk)),
				PopulationCaps: make([]float64, 0, len(chunk)),
				StartXs:        make([]int32, 0, len(chunk)),
				StartYs:        make([]int32, 0, len(chunk)),
				Sizes:          make([]int32, 0, len(chunk)),
			}

			for _, city := range chunk {
				params.CityIds = append(params.CityIds, city.CityID)
				params.Types = append(params.Types, string(city.Type))

				// sqlc will parse "" into NULL
				if city.Owner == nil {
					params.Owners = append(params.Owners, "")
				} else {
					params.Owners = append(params.Owners, *city.Owner)
				}

				params.Names = append(params.Names, city.Name)
				params.Populations = append(params.Populations, city.Population)
				params.PopulationCaps = append(params.PopulationCaps, city.PopulationCap)
				params.StartXs = append(params.StartXs, int32(city.StartX))
				params.StartYs = append(params.StartYs, int32(city.StartY))
				params.Sizes = append(params.Sizes, int32(city.Size))
			}

			if err := state.db.BatchUpdateCities(state.Ctx(), params); err != nil {
				slog.ErrorContext(state.Ctx(), "error batch updating cities", "idx", i, "error", err)
			}
		}

		userBatchSize := 5000
		users := make([]models.User, 0, len(state.userBuffer))
		for _, u := range state.userBuffer {
			users = append(users, u)
		}
		for i := 0; i < len(users); i += userBatchSize {
			end := min(i+userBatchSize, len(users))
			chunk := users[i:end]

			params := database.BatchUpdateUsersParams{
				UserIds: make([]string, 0, len(chunk)),
				Foods:   make([]int64, 0, len(chunk)),
				Golds:   make([]int64, 0, len(chunk)),
			}

			for _, user := range chunk {
				params.UserIds = append(params.UserIds, user.UserID)
				params.Foods = append(params.Foods, user.Food)
				params.Golds = append(params.Golds, user.Gold)
			}

			if err := state.db.BatchUpdateUsers(state.Ctx(), params); err != nil {
				slog.ErrorContext(state.Ctx(), "error batch updating users", "idx", i, "error", err)
			}
		}

		state.cityBuffer = make(map[string]models.City)
		state.userBuffer = make(map[string]models.User)
	}
}

func (state *databaseActor) startPeriodicOperation(ctx actor.Context) {
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
