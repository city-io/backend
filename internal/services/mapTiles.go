package services

import (
	"cityio/internal/actors"
	"cityio/internal/messages"
	"cityio/internal/models"

	"log"
)

func RestoreMapTile(tile models.MapTile) error {
	mapTilePID, err := actors.Spawn(&actors.MapTileActor{})
	if err != nil {
		log.Printf("Error spawning actor while restoring map tile: %s", err)
		return err
	}

	response, err := actors.Request[messages.CreateMapTileResponseMessage](system.Root, mapTilePID, messages.CreateMapTileMessage{
		Tile:    tile,
		Restore: true,
	})
	if err != nil {
		log.Printf("Error restoring map tile: %s", err)
		return err
	}
	if response.Error != nil {
		log.Printf("Error restoring map tile: %s", response.Error)
		return response.Error
	}

	addMapTilePIDResponse, err := actors.Request[messages.AddMapTilePIDResponseMessage](system.Root, actors.GetManagerPID(), messages.AddMapTilePIDMessage{
		X:   tile.X,
		Y:   tile.Y,
		PID: mapTilePID,
	})
	if err != nil {
		log.Printf("Error restoring map tile: %s", err)
		return err
	}
	if addMapTilePIDResponse.Error != nil {
		log.Printf("Error restoring map tile: %s", addMapTilePIDResponse.Error)
		return addMapTilePIDResponse.Error
	}

	return nil
}

func CreateMapTile(tile models.MapTile) error {
	mapTilePID, err := actors.Spawn(&actors.MapTileActor{})
	if err != nil {
		log.Printf("Error restoring map tile: %s", err)
		return err
	}

	response, err := actors.Request[messages.CreateMapTileResponseMessage](system.Root, mapTilePID, messages.CreateMapTileMessage{
		Tile:    tile,
		Restore: true,
	})
	if err != nil {
		log.Printf("Error restoring map tile: %s", err)
		return err
	}
	if response.Error != nil {
		log.Printf("Error restoring map tile: %s", response.Error)
		return response.Error
	}

	addMapTilePIDResponse, err := actors.Request[messages.AddMapTilePIDResponseMessage](system.Root, actors.GetManagerPID(), messages.AddMapTilePIDMessage{
		X:   tile.X,
		Y:   tile.Y,
		PID: mapTilePID,
	})
	if err != nil {
		log.Printf("Error restoring map tile: %s", err)
		return err
	}
	if addMapTilePIDResponse.Error != nil {
		log.Printf("Error restoring map tile: %s", addMapTilePIDResponse.Error)
		return addMapTilePIDResponse.Error
	}

	return nil
}

func GetMapTile(x int, y int) (models.MapTileOutput, error) {
	getMapTilePIDResponse, err := actors.Request[messages.GetMapTilePIDResponseMessage](system.Root, actors.GetManagerPID(), messages.GetMapTilePIDMessage{
		X: x,
		Y: y,
	})
	if err != nil {
		log.Printf("Error getting map tile: %s", err)
		return models.MapTileOutput{}, err
	}
	if getMapTilePIDResponse.PID == nil {
		log.Printf("Error getting map tile: %s", &messages.MapTileNotFoundError{X: x, Y: y})
		return models.MapTileOutput{}, &messages.MapTileNotFoundError{X: x, Y: y}
	}

	getMapTileResponse, err := actors.Request[messages.GetMapTileResponseMessage](system.Root, getMapTilePIDResponse.PID, messages.GetMapTileMessage{})
	if err != nil {
		log.Printf("Error getting map tile: %s", err)
		return models.MapTileOutput{}, err
	}

	if getMapTileResponse.City != nil && getMapTileResponse.City.Owner != "" {
		user, err := GetUser(getMapTileResponse.City.Owner)
		if err != nil {
			log.Printf("Error getting city owner: %s", err)
			return models.MapTileOutput{}, err
		}
		getMapTileResponse.City.Owner = user.Username
	}

	return models.MapTileOutput{
		X:        getMapTileResponse.Tile.X,
		Y:        getMapTileResponse.Tile.Y,
		City:     getMapTileResponse.City,
		Building: getMapTileResponse.Building,
		Armies:   getMapTileResponse.Armies,
	}, nil
}
