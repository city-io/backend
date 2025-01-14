package actors

import (
	"cityio/internal/database"
	"cityio/internal/messages"

	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"gorm.io/gorm"
)

var system *actor.ActorSystem
var managerPID *actor.PID

var systemOnce sync.Once
var managerPIDOnce sync.Once

type BaseActorInterface interface {
	Receive(ctx actor.Context)
	SetDb(db *gorm.DB)
	SetPIDActor(managerPID *actor.PID)
}

type BaseActor struct {
	db      *gorm.DB
	manager *actor.PID
}

func (b *BaseActor) Receive(ctx actor.Context) {
	// Should be overridden
}

func (b *BaseActor) SetDb(db *gorm.DB) {
	b.db = db
}

func (b *BaseActor) SetPIDActor(managerPID *actor.PID) {
	b.manager = managerPID
}

type ActorSystem interface {
	// Send(pid *actor.PID, message interface{})
	// Request(pid *actor.PID, message interface{})
	// RequestWithCustomSender(pid *actor.PID, message interface{}, sender *actor.PID)
	RequestFuture(pid *actor.PID, message interface{}, timeout time.Duration) *actor.Future
}

func initSystem() {
	system = actor.NewActorSystem()
}

func initManager() {
	var err error
	managerActor := PIDManagerActor{}
	managerPID, err = managerActor.Spawn()
	if err != nil {
		panic(err)
	}
}

func GetSystem() *actor.ActorSystem {
	systemOnce.Do(initSystem)
	return system
}

func GetManagerPID() *actor.PID {
	managerPIDOnce.Do(initManager)
	return managerPID
}

func (_actor *BaseActor) Spawn() (*actor.PID, error) {
	_actor.SetDb(database.GetDb())
	// TODO: update to use actual pid
	_actor.SetPIDActor(GetManagerPID())

	props := actor.PropsFromProducer(func() actor.Actor {
		return _actor
	})
	newPID := GetSystem().Root.Spawn(props)
	return newPID, nil
}

// func SendMessage(ctx ActorSystem, pid *actor.PID, message interface{}) {
// 	ctx.Send(pid, message)
// }

// func Respond(ctx actor.Context, message interface{}) {
// 	ctx.Respond(message)
// }

func Request[T any](ctx ActorSystem, pid *actor.PID, message interface{}) (*T, error) {
	future := ctx.RequestFuture(pid, message, 0)
	result, err := future.Result()
	if err != nil {
		return nil, err
	}

	if response, ok := result.(*T); ok {
		return response, nil
	}

	return nil, &messages.InvalidResponseTypeError{}
}
