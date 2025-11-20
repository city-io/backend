package ports

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"gorm.io/gorm"
)

type BaseActorInterface interface {
	Receive(ctx actor.Context)
	SetPIDActor(managerPID *actor.PID)
	SetDatabaseActor(databasePID *actor.PID)
	SetLog(log Logger)
}

type ActorSystem interface {
	RequestFuture(pid *actor.PID, message any, timeout time.Duration) *actor.Future
}

type ClusterProvider interface {
	Spawn(ac BaseActorInterface) (*actor.PID, error)
	DB() *gorm.DB
}
