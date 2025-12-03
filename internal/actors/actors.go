// Package actors provides definitions and implementations for various actor types used in the system.
package actors

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/controllers"
	"cityio/internal/logger"
	"cityio/internal/ports"
)

type baseActor struct {
	actor.Actor
	Log     logger.Logger
	Cluster ports.ClusterProvider
	Ctrls   *controllers.Controllers
}

func (b *baseActor) SetLog(log logger.Logger)                      { b.Log = log }
func (b *baseActor) SetCluster(cluster ports.ClusterProvider)      { b.Cluster = cluster }
func (b *baseActor) SetControllers(ctrls *controllers.Controllers) { b.Ctrls = ctrls }

type BaseActorInterface interface {
	ActorType() string
	Receive(ctx actor.Context)
	SetLog(log logger.Logger)
	SetCluster(cluster ports.ClusterProvider)
	SetControllers(ctrls *controllers.Controllers)
}
