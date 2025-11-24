package ports

import (
	"github.com/asynkron/protoactor-go/actor"
)

type ClusterProvider interface {
	Request(kind, identity string, message any) (any, error)
	RequestFuture(kind, identity string, message any) (actor.Future, error)
	Tell(kind, identity string, msg any) error
	DB() *actor.PID
}

type BaseActorInterface interface {
	ActorType() string
	Receive(ctx actor.Context)
	SetCluster(cluster ClusterProvider)
	SetLog(log Logger)
}
