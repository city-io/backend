package actors

import "github.com/asynkron/protoactor-go/actor"

type staticStructureImpl struct{}

func newStaticStructureImpl() buildingActorImpl { return &staticStructureImpl{} }

func (*staticStructureImpl) Create(actor.Context, *buildingActor)  {}
func (*staticStructureImpl) Destroy(actor.Context, *buildingActor) {}
func (*staticStructureImpl) Handle(actor.Context, *buildingActor)  {}
