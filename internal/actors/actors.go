// Package actors provides definitions and implementations for various actor types used in the system.
package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/ports"
)

type BaseActor struct {
	actor.Actor
	Cluster ports.ClusterProvider
	Log     ports.Logger
}

func (b *BaseActor) SetCluster(cluster ports.ClusterProvider) { b.Cluster = cluster }
func (b *BaseActor) SetLog(log ports.Logger)                  { b.Log = log }
