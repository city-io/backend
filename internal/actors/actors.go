// Package actors provides definitions and implementations for various actor types used in the system.
package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/ports"
)

type baseActor struct {
	actor.Actor
	Log     ports.Logger
	Cluster ports.ClusterProvider
	Ctrls   ports.Controllers
}

func (b *baseActor) SetLog(log ports.Logger)                  { b.Log = log }
func (b *baseActor) SetCluster(cluster ports.ClusterProvider) { b.Cluster = cluster }
func (b *baseActor) SetControllers(ctrls ports.Controllers)   { b.Ctrls = ctrls }
