package actors

import (
	"cityio/internal/constants"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

type ArmyActor struct {
	BaseActor
	Army models.Army

	OwnerPID *actor.PID

	armyOnce sync.Once

	ticker       *time.Ticker
	stopTickerCh chan struct{}
}

func (state *ArmyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {

	case messages.CreateArmyMessage:
		state.Army = msg.Army

		if !msg.Restore {
			// default coordinates to (-1, -1) to distinguish from (0, 0) tile
			if !state.Army.MarchActive {
				state.Army.FromX = -1
				state.Army.FromY = -1
				state.Army.ToX = -1
				state.Army.ToY = -1
			}

			ctx.Send(state.database, messages.CreateArmyMessage{
				Army: state.Army,
			})
		}

		getTilePIDResponse, err := Request[messages.GetMapTilePIDResponseMessage](ctx, GetManagerPID(), messages.GetMapTilePIDMessage{
			X: state.Army.TileX,
			Y: state.Army.TileY,
		})
		if err != nil {
			log.Printf("Error updating tile with new army: %s", err)
			return
		}
		if getTilePIDResponse.PID == nil {
			log.Printf("Error creating army: Map tile not found")
			return
		}

		ctx.Send(getTilePIDResponse.PID, messages.AddTileArmyMessage{
			ArmyPID: ctx.Self(),
			Army:    state.Army,
		})

		if state.Army.MarchActive {
			state.startTroopMovement(ctx)
		}
		ctx.Respond(messages.CreateArmyResponseMessage{
			Error: nil,
		})

	case messages.GetArmyMessage:
		ctx.Respond(messages.GetArmyResponseMessage{
			Army: state.Army,
		})

	case messages.UpdateArmyMessage:
		state.Army = msg.Army
		ctx.Send(state.database, messages.UpdateArmyMessage{
			Army: state.Army,
		})
		ctx.Respond(messages.UpdateArmyResponseMessage{
			Error: nil,
		})

	case messages.DeleteArmyMessage:
		ctx.Send(state.database, messages.DeleteArmyMessage{
			ArmyId: state.Army.ArmyId,
		})
		ctx.Respond(messages.DeleteArmyResponseMessage{
			Error: nil,
		})
		log.Printf("Shutting down ArmyActor for army: %s", state.Army.ArmyId)

		state.stopPeriodicOperation()
		ctx.Stop(ctx.Self())

	case messages.StartArmyMarchMessage:
		// init background march operation
		log.Printf("Army %s marching to (%d, %d)", state.Army.ArmyId, msg.X, msg.Y)
		state.Army.FromX = state.Army.TileX
		state.Army.FromY = state.Army.TileY
		state.Army.ToX = msg.X
		state.Army.ToY = msg.Y
		state.Army.MarchActive = true

		ctx.Send(state.database, messages.UpdateArmyMessage{
			Army: state.Army,
		})
		state.startTroopMovement(ctx)

	case messages.UpdateArmyTileMessage:
		// periodically update army position
		if !state.Army.MarchActive {
			state.stopPeriodicOperation()
			return
		}

		// move 1 at a time in the x direction, then y direction
		if state.Army.TileX < state.Army.ToX {
			state.Army.TileX++
		} else if state.Army.TileX > state.Army.ToX {
			state.Army.TileX--
		} else if state.Army.TileY < state.Army.ToY {
			state.Army.TileY++
		} else if state.Army.TileY > state.Army.ToY {
			state.Army.TileY--
		}
		log.Printf("Update: Army %s at (%d, %d)", state.Army.ArmyId, state.Army.TileX, state.Army.TileY)

		// TODO: send message to tile actor to update army list
		if state.Army.TileX == state.Army.ToX && state.Army.TileY == state.Army.ToY {
			state.stopPeriodicOperation()
			state.Army.MarchActive = false
			state.Army.FromX = -1
			state.Army.FromY = -1
			state.Army.ToX = -1
			state.Army.ToY = -1
			ctx.Send(state.database, messages.UpdateArmyMessage{
				Army: state.Army,
			})
		}
	}
}

func (state *ArmyActor) startTroopMovement(ctx actor.Context) {
	go func() {
		state.ticker = time.NewTicker(constants.TROOP_MOVEMENT_DURATION * time.Second)
		state.stopTickerCh = make(chan struct{})

		count := 0
		for {
			select {
			case <-state.ticker.C:
				// make periodic backups every 5 updates
				count++
				if count%5 == 0 {
					ctx.Send(state.database, messages.UpdateArmyMessage{
						Army: state.Army,
					})
				}
				ctx.Send(ctx.Self(), messages.UpdateArmyTileMessage{})
			case <-state.stopTickerCh:
				state.ticker.Stop()
				return
			}
		}
	}()
}

func (state *ArmyActor) stopPeriodicOperation() {
	if state.ticker != nil {
		close(state.stopTickerCh)
		state.ticker = nil
	}
}

func (state *ArmyActor) getTilePID() (*actor.PID, error) {
	getTilePIDResponse, err := Request[messages.GetMapTilePIDResponseMessage](system.Root, GetManagerPID(), messages.GetMapTilePIDMessage{
		X: state.Army.TileX,
		Y: state.Army.TileY,
	})
	if err != nil {
		log.Printf("Error restoring army: %s", err)
		return nil, err
	}
	if getTilePIDResponse.PID == nil {
		log.Printf("Error restoring army: Map tile not found")
		return nil, &messages.MapTileNotFoundError{X: state.Army.TileX, Y: state.Army.TileY}
	}
	return getTilePIDResponse.PID, nil
}

func (state *ArmyActor) getOwnerPID() (*actor.PID, error) {
	state.armyOnce.Do(func() {
		getOwnerPIDResponse, err := Request[messages.GetUserPIDResponseMessage](system.Root, GetManagerPID(), messages.GetUserPIDMessage{
			UserId: state.Army.Owner,
		})
		if err != nil {
			log.Printf("Error restoring army: %s", err)
			return
		}
		if getOwnerPIDResponse.PID == nil {
			log.Printf("Error restoring army: User not found")
			return
		}
		state.OwnerPID = getOwnerPIDResponse.PID
	})
	if state.OwnerPID == nil {
		return nil, &messages.UserNotFoundError{UserId: state.Army.Owner}
	}
	return state.OwnerPID, nil
}
