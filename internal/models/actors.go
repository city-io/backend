package models

import (
	"github.com/asynkron/protoactor-go/actor"

	"cityio/internal/ports"
)

type BaseActor struct {
	actor.Actor
	Manager  *actor.PID
	Database *actor.PID
	Log      ports.Logger
}

func (b *BaseActor) SetPIDActor(managerPID *actor.PID)       { b.Manager = managerPID }
func (b *BaseActor) SetDatabaseActor(databasePID *actor.PID) { b.Database = databasePID }
func (b *BaseActor) SetLog(log ports.Logger)                 { b.Log = log }
