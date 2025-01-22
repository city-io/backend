package actors

import (
	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"gorm.io/gorm"
)

type DatabaseActor struct {
	BaseActor
	db *gorm.DB

	userBuffer []models.User
	cityBuffer []models.City
	armyBuffer []models.Army

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func (state *DatabaseActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.InitDatabaseMessage:
		state.startPeriodicOperation(ctx)

	case messages.RegisterUserMessage:
		result := state.db.Create(&msg.User)
		if result.Error != nil {
			log.Printf("Error creating user in db: %s", result.Error)
		}
	case *messages.UpdateUserMessage:
		state.userBuffer = append(state.userBuffer, msg.User)
	case messages.DeleteUserMessage:
		result := state.db.Where("user_id = ?", msg.UserId).Delete(&models.User{})
		if result.Error != nil {
			log.Printf("Error deleting user in db: %s", result.Error)
		}

	case messages.CreateMapTileMessage:
		result := state.db.Create(&msg.Tile)
		if result.Error != nil {
			log.Printf("Error creating map tile in db: %s", result.Error)
		}

	case messages.CreateCityMessage:
		result := state.db.Create(&msg.City)
		if result.Error != nil {
			log.Printf("Error creating city in db: %s", result.Error)
		}
	case messages.DeleteCityMessage:
		result := state.db.Where("city_id = ?", msg.CityId).Delete(&models.City{})
		if result.Error != nil {
			log.Printf("Error deleting city in db: %s", result.Error)
		}
	case *messages.UpdateCityMessage:
		state.cityBuffer = append(state.cityBuffer, msg.City)

	case messages.CreateBuildingMessage:
		result := state.db.Create(&msg.Building)
		if result.Error != nil {
			log.Printf("Error creating building in db: %s", result.Error)
		}
	case messages.UpdateBuildingMessage:
		result := state.db.Save(&msg.Building)
		if result.Error != nil {
			log.Printf("Error updating building in db: %s", result.Error)
		}
	case messages.DeleteBuildingMessage:
		result := state.db.Where("building_id = ?", msg.BuildingId).Delete(&models.Building{})
		if result.Error != nil {
			log.Printf("Error deleting building in db: %s", result.Error)
		}

	case messages.CreateArmyMessage:
		result := state.db.Create(&msg.Army)
		if result.Error != nil {
			log.Printf("Error creating army in db: %s", result.Error)
		}
	case messages.UpdateArmyMessage:
		result := state.db.Save(&msg.Army)
		if result.Error != nil {
			log.Printf("Error updating army in db: %s", result.Error)
		}
	case messages.DeleteArmyMessage:
		result := state.db.Where("army_id = ?", msg.ArmyId).Delete(&models.Army{})
		if result.Error != nil {
			log.Printf("Error deleting army in db: %s", result.Error)
		}

	case messages.TrainTroopsMessage:
		result := state.db.Create(&msg.Training)
		if result.Error != nil {
			log.Printf("Error creating training in db: %s", result.Error)
		}
	case messages.DeleteTrainingMessage:
		result := state.db.Where("barracks_id = ?", msg.BarracksId).Delete(&models.Training{})
		if result.Error != nil {
			log.Printf("Error deleting training in db: %s", result.Error)
		}

	case messages.PeriodicOperationMessage:
		if len(state.userBuffer) > 0 {
			for _, user := range state.userBuffer {
				result := state.db.Save(&user)
				if result.Error != nil {
					log.Printf("Error updating user in db: %s", result.Error)
				}
			}
			state.userBuffer = make([]models.User, 0)
		}

		cityBatchSize := 5000
		if len(state.cityBuffer) > 0 {
			for i := 0; i < len(state.cityBuffer); i += cityBatchSize {
				end := i + cityBatchSize
				if end > len(state.cityBuffer) {
					end = len(state.cityBuffer)
				}
				if result := state.db.Save(state.cityBuffer[i:end]); result.Error != nil {
					log.Printf("Error creating cities: %s", result.Error)
				}
			}
			state.cityBuffer = make([]models.City, 0)
		}
	}
}

func (state *DatabaseActor) startPeriodicOperation(ctx actor.Context) {
	state.ticker = time.NewTicker(constants.DB_BACKUP_FREQUENCY * time.Second)

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
