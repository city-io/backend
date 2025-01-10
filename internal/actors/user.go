package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
	"gorm.io/gorm"
)

type UserActor struct {
	db   *gorm.DB
	User models.User
}

func NewUserActor(user models.User) *UserActor {
	actor := &UserActor{
		User: models.User{
			UserId:   user.UserId,
			Email:    user.Email,
			Username: user.Username,
			Password: user.Password,
		},
	}
	actor.init()
	return actor
}

func (actor *UserActor) init() {
	log.Printf("User actor with id %s has been spawned!", actor.User.UserId)
}

func (state *UserActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {

	case messages.GetUserMessage:
		ctx.Respond(state.User)
	}
}
