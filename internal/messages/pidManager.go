package messages

import "github.com/asynkron/protoactor-go/actor"

type InitPIDManagerMessage struct{}
type InitPIDManagerResponseMessage struct {
	Error error
}

type AddUserPIDMessage struct {
	UserId string
	PID    *actor.PID
}
type AddUserPIDResponseMessage struct {
	Error error
}

type GetUserPIDMessage struct {
	UserId string
}
type GetUserPIDResponseMessage struct {
	PID *actor.PID
}

type DeleteUserPIDMessage struct {
	UserId string
}
type DeleteUserPIDResponseMessage struct {
	Error error
}

type AddCityPIDMessage struct {
	CityId string
	PID    *actor.PID
}
type AddCityPIDResponseMessage struct {
	Error error
}

type GetCityPIDMessage struct {
	CityId string
}
type GetCityPIDResponseMessage struct {
	PID *actor.PID
}

type DeleteCityPIDMessage struct {
	CityId string
}
type DeleteCityPIDResponseMessage struct {
	Error error
}

type AddMapTilePIDMessage struct {
	X   int
	Y   int
	PID *actor.PID
}
type AddMapTilePIDResponseMessage struct {
	Error error
}

type GetMapTilePIDMessage struct {
	X int
	Y int
}
type GetMapTilePIDResponseMessage struct {
	PID *actor.PID
}

type AddArmyPIDMessage struct {
	ArmyId string
	PID    *actor.PID
}
type AddArmyPIDResponseMessage struct {
	Error error
}

type GetArmyPIDMessage struct {
	ArmyId string
}
type GetArmyPIDResponseMessage struct {
	PID *actor.PID
}

type DeleteArmyPIDMessage struct {
	ArmyId string
}
type DeleteArmyPIDResponseMessage struct {
	Error error
}
