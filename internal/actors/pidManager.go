package actors

import (
	"cityio/internal/messages"

	"github.com/asynkron/protoactor-go/actor"
)

type PIDManagerActor struct {
	BaseActor
	userPIDs     map[string]*actor.PID
	cityPIDs     map[string]*actor.PID
	mapTilePIDs  map[int]map[int]*actor.PID
	armyPIDs     map[string]*actor.PID
	buildingPIDs map[string]*actor.PID
}

func (state *PIDManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.InitPIDManagerMessage:
		state.userPIDs = make(map[string]*actor.PID)
		state.cityPIDs = make(map[string]*actor.PID)
		state.mapTilePIDs = make(map[int]map[int]*actor.PID)
		state.armyPIDs = make(map[string]*actor.PID)
		state.buildingPIDs = make(map[string]*actor.PID)
		ctx.Respond(messages.InitPIDManagerResponseMessage{
			Error: nil,
		})

	case messages.AddUserPIDMessage:
		state.userPIDs[msg.UserId] = msg.PID
		ctx.Respond(messages.AddUserPIDResponseMessage{
			Error: nil,
		})

	case messages.GetUserPIDMessage:
		ctx.Respond(messages.GetUserPIDResponseMessage{
			PID: state.userPIDs[msg.UserId],
		})

	case messages.DeleteUserPIDMessage:
		delete(state.userPIDs, msg.UserId)
		ctx.Respond(messages.DeleteUserPIDResponseMessage{
			Error: nil,
		})

	case messages.AddCityPIDMessage:
		state.cityPIDs[msg.CityId] = msg.PID
		ctx.Respond(messages.AddCityPIDResponseMessage{
			Error: nil,
		})

	case messages.GetCityPIDMessage:
		ctx.Respond(messages.GetCityPIDResponseMessage{
			PID: state.cityPIDs[msg.CityId],
		})

	case messages.DeleteCityPIDMessage:
		delete(state.cityPIDs, msg.CityId)
		ctx.Respond(messages.DeleteCityPIDResponseMessage{
			Error: nil,
		})

	case messages.AddMapTilePIDMessage:
		if _, ok := state.mapTilePIDs[msg.X]; !ok {
			state.mapTilePIDs[msg.X] = make(map[int]*actor.PID)
		}
		state.mapTilePIDs[msg.X][msg.Y] = msg.PID
		ctx.Respond(messages.AddMapTilePIDResponseMessage{
			Error: nil,
		})

	case messages.GetMapTilePIDMessage:
		if _, ok := state.mapTilePIDs[msg.X]; !ok {
			ctx.Respond(messages.GetMapTilePIDResponseMessage{
				PID: nil,
			})
		} else {
			ctx.Respond(messages.GetMapTilePIDResponseMessage{
				PID: state.mapTilePIDs[msg.X][msg.Y],
			})
		}

	case messages.AddArmyPIDMessage:
		state.armyPIDs[msg.ArmyId] = msg.PID
		ctx.Respond(messages.AddArmyPIDResponseMessage{
			Error: nil,
		})

	case messages.GetArmyPIDMessage:
		ctx.Respond(messages.GetArmyPIDResponseMessage{
			PID: state.armyPIDs[msg.ArmyId],
		})

	case messages.DeleteArmyPIDMessage:
		delete(state.armyPIDs, msg.ArmyId)
		ctx.Respond(messages.DeleteArmyPIDResponseMessage{
			Error: nil,
		})

	case messages.AddBuildingPIDMessage:
		state.buildingPIDs[msg.BuildingId] = msg.PID
		ctx.Respond(messages.AddBuildingPIDResponseMessage{
			Error: nil,
		})

	case messages.GetBuildingPIDMessage:
		ctx.Respond(messages.GetBuildingPIDResponseMessage{
			PID: state.buildingPIDs[msg.BuildingId],
		})

	case messages.DeleteBuildingPIDMessage:
		delete(state.buildingPIDs, msg.BuildingId)
		ctx.Respond(messages.DeleteBuildingPIDResponseMessage{
			Error: nil,
		})
	}
}
