package actors

import (
	"cityio/internal/database"
	"cityio/internal/messages"

	"log"
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
	actor.Actor
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
	props := actor.PropsFromProducer(func() actor.Actor {
		return &PIDManagerActor{}
	})
	managerPID = GetSystem().Root.Spawn(props)
	log.Printf("Spawned manager with PID: %s", managerPID)
}

func GetSystem() *actor.ActorSystem {
	systemOnce.Do(initSystem)
	return system
}

func GetManagerPID() *actor.PID {
	managerPIDOnce.Do(initManager)
	return managerPID
}

func Spawn[T BaseActorInterface](ac T) (*actor.PID, error) {
	return SpawnBase(func() actor.Actor {
		return ac
	})
}

func SpawnBase(newActor func() actor.Actor) (*actor.PID, error) {
	props := actor.PropsFromProducer(func() actor.Actor {
		a := newActor()
		if baseActor, ok := a.(BaseActorInterface); ok {
			baseActor.SetDb(database.GetDb())
			baseActor.SetPIDActor(GetManagerPID())
		}
		return a
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
	future := ctx.RequestFuture(pid, message, time.Second)
	result, err := future.Result()
	if err != nil {
		return nil, err
	}

	if response, ok := result.(T); ok {
		return &response, nil
	}
	return nil, &messages.InvalidResponseTypeError{}
}
