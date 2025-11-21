package ports

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"gorm.io/gorm"
)

type BaseActorInterface interface {
	Receive(ctx actor.Context)
	SetDatabaseActor(databasePID *actor.PID)
	SetLog(log Logger)
}

type ActorSystem interface {
	RequestFuture(pid *actor.PID, message any, timeout time.Duration) *actor.Future
}

type ClusterProvider interface {
	DB() *gorm.DB
	Request(identity string, kind string, message any) (any, error)
	Tell(kind, identity string, msg any) error
}
