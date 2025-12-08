package cluster

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/cluster"
	"github.com/asynkron/protoactor-go/cluster/clusterproviders/consul"
	"github.com/asynkron/protoactor-go/cluster/clusterproviders/test"
	"github.com/asynkron/protoactor-go/cluster/identitylookup/disthash"
	"github.com/asynkron/protoactor-go/remote"

	"cityio/internal/actors"
	"cityio/internal/constants"
	"cityio/internal/database"
	"cityio/internal/logger"
	"cityio/internal/messages"
)

type ClusterProvider struct {
	log         logger.Logger
	system      *actor.ActorSystem
	cluster     *cluster.Cluster
	databasePID *actor.PID
}

func NewRuntime(log logger.Logger, db database.Querier) *ClusterProvider {
	system := actor.NewActorSystem()

	databaseProps := actor.PropsFromProducer(func() actor.Actor {
		ac := actors.NewDatabaseActor(db)
		ac.SetLog(log)
		return ac
	})
	databasePID := system.Root.Spawn(databaseProps)
	system.Root.Send(databasePID, messages.InitDatabaseMessage{})

	cp := &ClusterProvider{
		log:         log,
		system:      system,
		cluster:     nil,
		databasePID: databasePID,
	}

	spawn := func(newActor func() actors.BaseActorInterface) actor.Producer {
		return func() actor.Actor {
			ac := newActor()
			ac.SetLog(log.With("actor", ac.ActorType()))
			ac.SetCluster(cp)
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
	if constants.Environment == "development" {
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
	cp.system.Root.Send(pid, msg)
	return nil
}

func (cp *ClusterProvider) DB() *actor.PID {
	return cp.databasePID
}

func (cp *ClusterProvider) RequestDBFuture(message any) actor.Future {
	return cp.system.Root.RequestFuture(cp.databasePID, message, constants.ActorTimeoutDuration*time.Second)
}

// shouldn't generally be used
func (cp *ClusterProvider) SendDB(message any) {
	cp.system.Root.Send(cp.databasePID, message)
}
