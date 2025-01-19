package actors

import (
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

var system *actor.ActorSystem
var managerPID *actor.PID
var databasePID *actor.PID

var systemOnce sync.Once
var managerPIDOnce sync.Once
var databasePIDOnce sync.Once

type BaseActorInterface interface {
	Receive(ctx actor.Context)
	SetPIDActor(managerPID *actor.PID)
	SetDatabaseActor(databasePID *actor.PID)
}

type BaseActor struct {
	actor.Actor
	manager  *actor.PID
	database *actor.PID
}

func (b *BaseActor) Receive(ctx actor.Context) {
	// Should be overridden
}

func (b *BaseActor) SetPIDActor(managerPID *actor.PID) {
	b.manager = managerPID
}

func (b *BaseActor) SetDatabaseActor(databasePID *actor.PID) {
	b.database = databasePID
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

func initDatabaseActor() {
	props := actor.PropsFromProducer(func() actor.Actor {
		return &DatabaseActor{
			db:         database.GetDb(),
			userBuffer: make([]models.User, 0),
			cityBuffer: make([]models.City, 0),
			armyBuffer: make([]models.Army, 0),
		}
	})
	databasePID = GetSystem().Root.Spawn(props)
	log.Printf("Spawned database actor with PID: %s", managerPID)
	system.Root.Send(databasePID, messages.InitDatabaseMessage{})
}

func GetSystem() *actor.ActorSystem {
	systemOnce.Do(initSystem)
	return system
}

func GetManagerPID() *actor.PID {
	managerPIDOnce.Do(initManager)
	return managerPID
}

func GetDatabasePID() *actor.PID {
	databasePIDOnce.Do(initDatabaseActor)
	return databasePID
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
			baseActor.SetPIDActor(GetManagerPID())
			baseActor.SetDatabaseActor(GetDatabasePID())
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
