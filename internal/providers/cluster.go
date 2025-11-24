package providers

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
	"cityio/internal/controllers"
	"cityio/internal/database"
	"cityio/internal/messages"
	"cityio/internal/ports"
)

type clusterProvider struct {
	log         ports.Logger
	system      *actor.ActorSystem
	cluster     *cluster.Cluster
	databasePID *actor.PID
}

func NewRuntime(log ports.Logger, db database.Querier) (ports.ClusterProvider, ports.Controllers) {
	system := actor.NewActorSystem()

	databaseProps := actor.PropsFromProducer(func() actor.Actor {
		ac := actors.NewDatabaseActor(db)
		ac.SetLog(log)
		return ac
	})
	databasePID := system.Root.Spawn(databaseProps)
	system.Root.Send(databasePID, messages.InitDatabaseMessage{})

	cp := &clusterProvider{
		log:         log,
		system:      system,
		cluster:     nil,
		databasePID: databasePID,
	}
	ctrls := controllers.NewControllers(cp, log)

	spawn := func(newActor func() ports.BaseActorInterface) actor.Producer {
		return func() actor.Actor {
			ac := newActor()
			ac.SetDatabaseActor(databasePID)
			ac.SetLog(log.With("actor", ac.ActorType()))
			ac.SetCluster(cp)
			return ac
		}
	}
	kinds := []*cluster.Kind{
		cluster.NewKind("user", actor.PropsFromProducer(spawn(actors.NewUserActor))),
		cluster.NewKind("city", actor.PropsFromProducer(spawn(actors.NewCityActor))),
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

	clusterConfig := cluster.Configure("cityio-cluster", provider, lookup, remoteConfig, cluster.WithKinds(kinds...))
	cl := cluster.New(system, clusterConfig)
	cp.cluster = cl
	cl.StartMember()

	return cp, ctrls
}

func (cp *clusterProvider) Request(kind, identity string, message any) (any, error) {
	return cp.cluster.Request(identity, kind, message)
}

func (cp *clusterProvider) RequestFuture(kind, identity string, message any) (actor.Future, error) {
	return cp.cluster.RequestFuture(
		identity,
		kind,
		message,
		cluster.WithTimeout(constants.ActorTimeoutDuration*time.Second),
	)
}

func (cp *clusterProvider) Tell(kind, identity string, msg any) error {
	pid := cp.cluster.Get(identity, kind)
	cp.system.Root.Send(pid, msg)
	return nil
}
