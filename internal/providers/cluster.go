package providers

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/cluster"
	"github.com/asynkron/protoactor-go/cluster/clusterproviders/consul"
	"github.com/asynkron/protoactor-go/cluster/identitylookup/disthash"
	"github.com/asynkron/protoactor-go/remote"
	"gorm.io/gorm"

	"cityio/internal/actors"
	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/ports"
)

type clusterProvider struct {
	log         ports.Logger
	system      *actor.ActorSystem
	cluster     *cluster.Cluster
	managerPID  *actor.PID
	databasePID *actor.PID
	db          *gorm.DB
}

func NewClusterProvider(log ports.Logger, db *gorm.DB) ports.ClusterProvider {
	system := actor.NewActorSystem()

	var kinds []*cluster.Kind
	kinds = append(kinds, cluster.NewKind("User", actor.PropsFromProducer(NewUserActor)))
	kinds = append(kinds, cluster.NewKind("City", actor.PropsFromProducer(NewCityActor)))

	remoteConfig := remote.Configure("127.0.0.1", 8090)
	provider, err := consul.New()
	if err != nil {
		panic(err)
	}
	lookup := disthash.New()

	clusterConfig := cluster.Configure("cityio-cluster", provider, lookup, remoteConfig, cluster.WithKinds(kinds...))
	cl := cluster.New(system, clusterConfig)
	cl.StartMember()

	managerProps := actor.PropsFromProducer(actors.NewPIDManager)
	databaseProps := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewDatabaseActor(db)
	})
	return &clusterProvider{
		log:         log,
		system:      actor.NewActorSystem(),
		managerPID:  system.Root.Spawn(managerProps),
		databasePID: system.Root.Spawn(databaseProps),
		db:          db,
	}
}

func (a *clusterProvider) DB() *gorm.DB {
	return a.db
}

func (a *clusterProvider) Spawn(ac ports.BaseActorInterface) (*actor.PID, error) {
	props := actor.PropsFromProducer(func() actor.Actor {
		ac.SetPIDActor(a.managerPID)
		ac.SetDatabaseActor(a.databasePID)
		ac.SetLog(a.log)
		return ac
	})
	newPID := a.system.Root.Spawn(props)
	return newPID, nil
}

func Request[T any](ctx ports.ActorSystem, pid *actor.PID, message any) (*T, error) {
	future := ctx.RequestFuture(pid, message, constants.ACTOR_TIMEOUT_DURATION*time.Second)
	result, err := future.Result()
	if err != nil {
		return nil, err
	}

	if response, ok := result.(T); ok {
		return &response, nil
	}
	return nil, &messages.InvalidResponseTypeError{}
}
