// Package actors provides definitions and implementations for various actor types used in the system.
package actors

import (
	"context"

	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/ports"
)

type baseActor struct {
	actor.Actor
	ctx     context.Context
	Cluster ports.ClusterProvider
}

// SetContext stores the base logging context for the actor. Attributes carried
// on this context (such as the actor type) are emitted by every slog call the
// actor makes.
func (b *baseActor) SetContext(ctx context.Context)           { b.ctx = ctx }
func (b *baseActor) SetCluster(cluster ports.ClusterProvider) { b.Cluster = cluster }

// Ctx returns the actor's base logging context, falling back to a background
// context when none has been set.
func (b *baseActor) Ctx() context.Context {
	if b.ctx == nil {
		return context.Background()
	}
	return b.ctx
}

type BaseActorInterface interface {
	ActorType() string
	Receive(ctx actor.Context)
	SetContext(ctx context.Context)
	SetCluster(cluster ports.ClusterProvider)
}
