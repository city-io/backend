package actors

import (
	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"math"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
)

type BarracksActor struct {
	BuildingActor
	Training *models.Training
}

func (state *BarracksActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateBuildingMessage:
		state.Building = msg.Building
		if !msg.Restore {
			ctx.Send(state.database, messages.CreateBuildingMessage{
				Building: state.Building,
			})
		}
		ctx.Respond(messages.CreateBuildingResponseMessage{
			Error: nil,
		})

	case messages.UpgradeBuildingMessage:
		ctx.Respond(messages.UpgradeBuildingResponseMessage{
			Error: state.upgradeBuilding(ctx),
		})

	case messages.RestoreTrainingMessage:
		log.Printf("Restoring training %+v", msg.Training)
		state.Training = &msg.Training
		if state.Training.End.Before(time.Now()) {
			state.completeTraining(ctx)
		} else {
			go state.backgroundTrain(ctx)
		}
		ctx.Respond(messages.RestoreTrainingResponseMessage{
			Error: nil,
		})

	case messages.TrainTroopsMessage:
		if state.Training != nil {
			ctx.Respond(messages.TrainTroopsResponseMessage{
				Error: &messages.TrainingAlreadyExistsError{BarracksId: state.Training.BarracksId},
			})
			return
		}
		endTime := time.Now().Add(time.Second * constants.TROOP_TRAINING_DURATION)
		if state.Building.BuildingId != msg.Training.BarracksId {
			ctx.Respond(messages.TrainTroopsResponseMessage{
				Error: &messages.UnknownError{Message: "Requested barracks id does not match this building"},
			})
			return
		}
		state.Training = &models.Training{
			BarracksId: state.Building.BuildingId,
			Size:       msg.Training.Size,
			DeployTo:   msg.Training.DeployTo,
			End:        endTime,
		}
		log.Printf("Spawning traning of %d troops", state.Training.Size)
		ctx.Send(GetDatabasePID(), messages.TrainTroopsMessage{
			Training: *state.Training,
		})
		go state.backgroundTrain(ctx)
		ctx.Respond(messages.TrainTroopsResponseMessage{
			Error: nil,
		})

	case messages.GetBuildingMessage:
		ctx.Respond(messages.GetBuildingResponseMessage{
			Building: state.Building,
		})

	case messages.DeleteBuildingMessage:
		state.deleteBuilding(ctx)
	}
}

func (state *BarracksActor) backgroundTrain(ctx actor.Context) {
	time.Sleep(time.Until(state.Training.End))
	state.completeTraining(ctx)
}

func (state *BarracksActor) completeTraining(ctx actor.Context) {
	log.Printf("Training complete for %+v", state.Training)
	ctx.Send(GetDatabasePID(), messages.DeleteTrainingMessage{
		BarracksId: state.Training.BarracksId,
	})
	ownerId, err := state.getOwnerId()
	if err != nil {
		log.Printf("Error completing training: %s", err)
		return
	}

	if state.Training.DeployTo != "" && state.Training.DeployTo != state.Building.CityId {
		// TODO: do ownership verification checks of deployment city here at deploy time, not in api
		getDeployCityPIDResponse, err := Request[messages.GetCityPIDResponseMessage](system.Root, GetManagerPID(), messages.GetCityPIDMessage{
			CityId: state.Training.DeployTo,
		})
		// TODO: simplify error flow lol
		if err != nil {
			log.Printf("Error fetching deployment city pid after training, defaulting to same city barracks")
			state.createArmy(ctx, models.Army{
				TileX: state.Building.X,
				TileY: state.Building.Y,
				Owner: ownerId,
				Size:  state.Training.Size,
			})
		}
		if getDeployCityPIDResponse.PID == nil {
			log.Printf("Error fetching deployment city pid after training, defaulting to same city barracks")
			state.createArmy(ctx, models.Army{
				TileX: state.Building.X,
				TileY: state.Building.Y,
				Owner: ownerId,
				Size:  state.Training.Size,
			})
		}

		var getDeployCityResponse *messages.GetCityResponseMessage
		getDeployCityResponse, err = Request[messages.GetCityResponseMessage](ctx, getDeployCityPIDResponse.PID, messages.GetCityMessage{})
		if err != nil {
			log.Printf("Error fetching deployment city after training, defaulting to same city barracks")
			state.createArmy(ctx, models.Army{
				TileX: state.Building.X,
				TileY: state.Building.Y,
				Owner: ownerId,
				Size:  state.Training.Size,
			})
		}

		cityX := getDeployCityResponse.City.StartX + int(math.Floor(float64(getDeployCityResponse.City.Size)/2))
		cityY := getDeployCityResponse.City.StartY + int(math.Floor(float64(getDeployCityResponse.City.Size)/2))
		log.Printf("Deploying to city at (%d, %d)", cityX, cityY)
		state.createArmy(ctx, models.Army{
			TileX: state.Building.X,
			TileY: state.Building.Y,
			Owner: ownerId,
			Size:  state.Training.Size,

			FromX:       state.Building.X,
			FromY:       state.Building.Y,
			ToX:         cityX,
			ToY:         cityY,
			MarchActive: true,
		})
	} else {
		cityPID := state.getCityPID()
		if cityPID == nil {
			log.Printf("Error fetching city pid after training, spawning in city barracks")
			state.createArmy(ctx, models.Army{
				TileX: state.Building.X,
				TileY: state.Building.Y,
				Owner: ownerId,
				Size:  state.Training.Size,
			})
		}

		getCityResponse, err := Request[messages.GetCityResponseMessage](ctx, cityPID, messages.GetCityMessage{})
		if err != nil {
			log.Printf("Error fetching city after training, spawning in city barracks")
			state.createArmy(ctx, models.Army{
				TileX: state.Building.X,
				TileY: state.Building.Y,
				Owner: ownerId,
				Size:  state.Training.Size,
			})
		}
		state.createArmy(ctx, models.Army{
			TileX: getCityResponse.City.StartX + int(math.Floor(float64(getCityResponse.City.Size)/2)),
			TileY: getCityResponse.City.StartY + int(math.Floor(float64(getCityResponse.City.Size)/2)),
			Owner: ownerId,
			Size:  state.Training.Size,
		})
	}
	state.Training = nil
}

func (state *BarracksActor) createArmy(ctx actor.Context, army models.Army) error {
	userPID := state.getUserPID()
	if userPID == nil {
		log.Printf("Error creating army: User not found")
		return &messages.UserNotFoundError{UserId: army.Owner}
	}

	armyPID, err := Spawn(&ArmyActor{})

	army.ArmyId = uuid.New().String()
	createArmyResponse, err := Request[messages.CreateArmyResponseMessage](ctx, armyPID, messages.CreateArmyMessage{
		Army:    army,
		Restore: false,
	})
	if err != nil {
		log.Printf("Error creating army: %s", err)
		return err
	}
	if createArmyResponse.Error != nil {
		log.Printf("Error creating army: %s", createArmyResponse.Error)
		return createArmyResponse.Error
	}

	addUserArmyResponse, err := Request[messages.AddUserArmyResponseMessage](ctx, userPID, messages.AddUserArmyMessage{
		ArmyId:  army.ArmyId,
		ArmyPID: armyPID,
	})
	if err != nil {
		log.Printf("Error creating army: %s", err)
		return err
	}
	if addUserArmyResponse.Error != nil {
		log.Printf("Error creating army: %s", addUserArmyResponse.Error)
		return addUserArmyResponse.Error
	}

	addArmyPIDResponse, err := Request[messages.AddArmyPIDResponseMessage](ctx, GetManagerPID(), messages.AddArmyPIDMessage{
		ArmyId: army.ArmyId,
		PID:    armyPID,
	})
	if err != nil {
		log.Printf("Error creating army: %s", err)
		return err
	}
	if addArmyPIDResponse.Error != nil {
		log.Printf("Error creating army: %s", addArmyPIDResponse.Error)
		return addArmyPIDResponse.Error
	}

	log.Printf("Created army at (%d, %d) of size %d", army.TileX, army.TileY, army.Size)
	return nil
}
