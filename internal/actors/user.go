package actors

import (
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"

	"github.com/asynkron/protoactor-go/actor"
	"gorm.io/gorm"
)

type UserActor struct {
	Db   *gorm.DB
	User models.User
}

func NewUserActor(user models.User, db *gorm.DB) *UserActor {
	actor := &UserActor{
		User: models.User{
			UserId:   user.UserId,
			Email:    user.Email,
			Username: user.Username,
			Password: user.Password,
		},
		Db: db,
	}
	actor.init()
	return actor
}

func (actor *UserActor) init() {
	result := actor.Db.Create(&actor.User)
	if result.Error != nil {
		log.Printf("Error creating user: %s", result.Error)
		return
	}
	log.Printf("User actor with id %s has been spawned!", actor.User.UserId)
}

func (state *UserActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {

	case messages.GetUserMessage:
		ctx.Respond(state.User)
	}
}
