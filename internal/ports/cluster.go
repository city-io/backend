package ports

import "github.com/asynkron/protoactor-go/actor"

type ClusterProvider interface {
	Request(kind, identity string, message any) (any, error)
	RequestFuture(kind, identity string, message any) (actor.Future, error)
	Tell(kind, identity string, msg any) error
	DB() *actor.PID
	RequestDBFuture(message any) actor.Future
	SendDB(message any)
}
