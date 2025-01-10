package actors

import (
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

var system *actor.ActorSystem
var once sync.Once

func initSystem() {
	system = actor.NewActorSystem()
}

func GetSystem() *actor.ActorSystem {
	once.Do(initSystem)
	return system
}
