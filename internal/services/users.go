package services

import (
	"cityio/internal/actors"
	"cityio/internal/messages"
	"cityio/internal/models"
	"cityio/internal/state"

	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/google/uuid"
)

func RestoreUser(system *actor.ActorSystem, user models.User) {
	log.Printf("Restoring user: %s", user.UserId)
	props := actor.PropsFromProducer(func() actor.Actor {
		return &actors.UserActor{
			User: user,
		}
	})
	newPID := system.Root.Spawn(props)
	state.AddUserPID(user.UserId, newPID)
}

func RegisterUser(system *actor.ActorSystem, user models.UserInput) (string, error) {
	userId := uuid.New().String()
	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewUserActor(models.User{
			UserId:   userId,
			Email:    user.Email,
			Username: user.Username,
			Password: user.Password,
		})
	})
	newPID := system.Root.Spawn(props)
	state.AddUserPID(userId, newPID)
	return userId, nil
}

func GetUser(system *actor.ActorSystem, userId string) (models.User, error) {
	userPID, exists := state.GetUserPID(userId)
	if !exists {
		return models.User{}, &messages.UserNotFoundError{UserId: userId}
	}

	future := system.Root.RequestFuture(userPID, messages.GetUserMessage{}, time.Second*2)
	response, err := future.Result()
	if err != nil {
		return models.User{}, err
	}

	user, ok := response.(models.User)
	if !ok {
		return models.User{}, &messages.UserNotFoundError{UserId: userId}
	}

	return user, nil
}
