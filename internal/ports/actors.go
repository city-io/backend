package ports

import (
	"github.com/asynkron/protoactor-go/actor"
)

type ClusterProvider interface {
	Request(kind, identity string, message any) (any, error)
	RequestFuture(kind, identity string, message any) (actor.Future, error)
	Tell(kind, identity string, msg any) error
	DB() *actor.PID
	RequestDBFuture(message any) actor.Future
	SendDB(message any) // shouldn't need to be used
}

type BaseActorInterface interface {
	ActorType() string
	Receive(ctx actor.Context)
	SetLog(log Logger)
	SetCluster(cluster ClusterProvider)
	SetControllers(ctrls Controllers)
}
