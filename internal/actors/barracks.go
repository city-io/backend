package actors

import (
	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
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
			state.createBuilding(ctx)
		}
		ctx.Respond(messages.CreateBuildingResponseMessage{
			Error: nil,
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
		ctx.Send(GetDatabasePID(), messages.TrainTroopsMessage{
			Training: *state.Training,
		})
		log.Printf("Spawning traning of %+v", state.Training)
		go state.backgroundTrain(ctx)
		ctx.Respond(messages.TrainTroopsResponseMessage{
			Error: nil,
		})

	case messages.UpdateBuildingTilePIDMessage:
		state.MapTilePID = msg.TilePID

	case messages.GetBuildingMessage:
		state.getBuilding(ctx)

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
	ownerId, err := state.getOwnerId()
	if err != nil {
		log.Printf("Error completing training: %s", err)
		return
	}

	state.createArmy(ctx, models.Army{
		TileX: state.Building.X,
		TileY: state.Building.Y,
		Owner: ownerId,
		Size:  state.Training.Size,
	})

	if state.Training.DeployTo != "" {
		// TODO: Spawn a march order to new deployment city
	}
	ctx.Send(GetDatabasePID(), messages.DeleteTrainingMessage{
		BarracksId: state.Training.BarracksId,
	})
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
		Army:     army,
		OwnerPID: userPID,
		Restore:  false,
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

	getTilePIDResponse, err := Request[messages.GetMapTilePIDResponseMessage](ctx, GetManagerPID(), messages.GetMapTilePIDMessage{
		X: army.TileX,
		Y: army.TileY,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return err
	}
	if getTilePIDResponse.PID == nil {
		log.Printf("Error restoring army: Map tile not found")
		return &messages.MapTileNotFoundError{X: army.TileX, Y: army.TileY}
	}

	// TODO: replace with better way of storing armies in tiles
	addTileArmyPIDResponse, err := Request[messages.AddTileArmyResponseMessage](ctx, getTilePIDResponse.PID, messages.AddTileArmyMessage{
		ArmyPID: armyPID,
		Army:    army,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return err
	}
	if addTileArmyPIDResponse.Error != nil {
		log.Printf("Error restoring army: %s", addTileArmyPIDResponse.Error)
		return addTileArmyPIDResponse.Error
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
