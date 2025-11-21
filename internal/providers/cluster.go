package providers

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/cluster"
	"github.com/asynkron/protoactor-go/cluster/clusterproviders/consul"
	"github.com/asynkron/protoactor-go/cluster/clusterproviders/test"
	"github.com/asynkron/protoactor-go/cluster/identitylookup/disthash"
	"github.com/asynkron/protoactor-go/remote"
	"gorm.io/gorm"

	"cityio/internal/actors"
	"cityio/internal/constants"
	"cityio/internal/ports"
)

type clusterProvider struct {
	log         ports.Logger
	system      *actor.ActorSystem
	cluster     *cluster.Cluster
	databasePID *actor.PID
	db          *gorm.DB
}

func NewClusterProvider(log ports.Logger, db *gorm.DB) ports.ClusterProvider {
	system := actor.NewActorSystem()

	databaseProps := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewDatabaseActor(db)
	})
	databasePID := system.Root.Spawn(databaseProps)

	spawn := func(newActor func() ports.BaseActorInterface) actor.Producer {
		return func() actor.Actor {
			ac := newActor()
			ac.SetDatabaseActor(databasePID)
			ac.SetLog(log)
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
	cl.StartMember()

	return &clusterProvider{
		log:         log,
		system:      system,
		cluster:     cl,
		databasePID: databasePID,
		db:          db,
	}
}

func (cp *clusterProvider) DB() *gorm.DB {
	return cp.db
}

func (cp *clusterProvider) Request(identity string, kind string, message any) (any, error) {
	return cp.cluster.Request(identity, kind, message)
}

func (cp *clusterProvider) Tell(kind, identity string, msg any) error {
	pid := cp.cluster.Get(kind, identity)
	cp.system.Root.Send(pid, msg)
	return nil
}
