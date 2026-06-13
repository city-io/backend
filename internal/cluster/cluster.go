package cluster

import (
	"context"
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/cluster"
	"github.com/asynkron/protoactor-go/cluster/clusterproviders/consul"
	"github.com/asynkron/protoactor-go/cluster/clusterproviders/test"
	"github.com/asynkron/protoactor-go/cluster/identitylookup/disthash"
	"github.com/asynkron/protoactor-go/remote"

	"cityio/internal/actors"
	"cityio/internal/constants"
	"cityio/internal/logger"
	"cityio/internal/ports"
)

type ClusterProvider struct {
	system  *actor.ActorSystem
	cluster *cluster.Cluster
}

func NewRuntime(ctx context.Context, store ports.Store, environment string) *ClusterProvider {
	system := actor.NewActorSystem()

	cp := &ClusterProvider{
		system:  system,
		cluster: nil,
	}

	spawn := func(newActor func() actors.BaseActorInterface) actor.Producer {
		return func() actor.Actor {
			ac := newActor()
			ac.SetContext(logger.With(ctx, "actor", ac.ActorType()))
			ac.SetCluster(cp)
			ac.SetStore(store)
			return ac
		}
	}
	kinds := []*cluster.Kind{
		cluster.NewKind("user", actor.PropsFromProducer(spawn(actors.NewUserActor))),
		cluster.NewKind("city", actor.PropsFromProducer(spawn(actors.NewCityActor))),
		cluster.NewKind("tile", actor.PropsFromProducer(spawn(actors.NewTileActor))),
		cluster.NewKind("building", actor.PropsFromProducer(spawn(actors.NewBuildingActor))),
	}

	remoteConfig := remote.Configure("127.0.0.1", 8090)
	lookup := disthash.New()

	var provider cluster.ClusterProvider
	if environment != "production" {
		testagent := test.NewInMemAgent()
		provider = test.NewTestProvider(testagent)
	} else {
		// TODO: switch to kubernetes provider
		var err error
		provider, err = consul.New()
		if err != nil {
			panic(err)
		}
	}

	clusterConfig := cluster.Configure("cityio-cluster", provider, lookup, remoteConfig, cluster.WithKinds(kinds...), cluster.WithRequestLog(false))
	cl := cluster.New(system, clusterConfig)
	cp.cluster = cl
	cl.StartMember()

	return cp
}

func (cp *ClusterProvider) Request(kind, identity string, message any) (any, error) {
	return cp.cluster.Request(identity, kind, message)
}

func (cp *ClusterProvider) RequestFuture(kind, identity string, message any) (actor.Future, error) {
	return cp.cluster.RequestFuture(
		identity,
		kind,
		message,
		cluster.WithTimeout(constants.ActorTimeoutDuration*time.Second),
	)
}

func (cp *ClusterProvider) Tell(kind, identity string, msg any) error {
	pid := cp.cluster.Get(identity, kind)
	if pid == nil {
		return fmt.Errorf("could not resolve actor %s/%s", kind, identity)
	}
	cp.system.Root.Send(pid, msg)
	return nil
}
