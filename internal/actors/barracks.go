package actors

import (
	"github.com/asynkron/protoactor-go/actor"
)

type barracksImpl struct{}

func newBarracksImpl() buildingActorImpl {
	return &barracksImpl{}
}

func (b *barracksImpl) Create(ctx actor.Context, state *buildingActor)  {}
func (b *barracksImpl) Destroy(ctx actor.Context, state *buildingActor) {}
func (b *barracksImpl) Handle(ctx actor.Context, state *buildingActor)  {}
