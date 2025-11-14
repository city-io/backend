package models

import "github.com/asynkron/protoactor-go/actor"

type BaseActor struct {
	actor.Actor
	Manager  *actor.PID
	Database *actor.PID
}

func (b *BaseActor) SetPIDActor(managerPID *actor.PID)       { b.Manager = managerPID }
func (b *BaseActor) SetDatabaseActor(databasePID *actor.PID) { b.Database = databasePID }
