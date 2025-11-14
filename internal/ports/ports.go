package ports

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

type BaseActorInterface interface {
	Receive(ctx actor.Context)
	SetPIDActor(managerPID *actor.PID)
	SetDatabaseActor(databasePID *actor.PID)
}

type ActorSystem interface {
	RequestFuture(pid *actor.PID, message interface{}, timeout time.Duration) *actor.Future
}
