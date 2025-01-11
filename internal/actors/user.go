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

func NewUserActor(db *gorm.DB) *UserActor {
	actor := &UserActor{
		Db: db,
	}
	return actor
}

func (state *UserActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.RegisterUserMessage:
		state.User = msg.User
		if !msg.Restore {
			err := state.createUser()
			ctx.Respond(messages.RegisterUserResponseMessage{
				Error: err,
			})
		}

	case messages.GetUserMessage:
		ctx.Respond(messages.GetUserResponseMessage{
			User: state.User,
		})

	case messages.DeleteUserMessage:
		result := state.Db.Delete(&state.User)
		if result.Error != nil {
			log.Printf("Error deleting user: %s", result.Error)
		}
		ctx.Respond(messages.DeleteUserResponseMessage{
			Error: result.Error,
		})
		log.Printf("Shutting down UserActor for user: %s", state.User.Username)
		ctx.Stop(ctx.Self())
	}
}

func (state *UserActor) createUser() error {
	result := state.Db.Create(&state.User)
	if result.Error != nil {
		log.Printf("Error creating user: %s", result.Error)
		return result.Error
	}
	return nil
}
