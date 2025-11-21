package models

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/ports"
)

type BaseActor struct {
	actor.Actor
	Database *actor.PID
	Cluster  ports.ClusterProvider
	Log      ports.Logger
}

func (b *BaseActor) SetDatabaseActor(databasePID *actor.PID)  { b.Database = databasePID }
func (b *BaseActor) SetCluster(cluster ports.ClusterProvider) { b.Cluster = cluster }
func (b *BaseActor) SetLog(log ports.Logger)                  { b.Log = log }
