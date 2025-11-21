package main

import (
	"cityio/internal/api"
	"cityio/internal/database"
	"cityio/internal/logger"
	"cityio/internal/providers"
	"cityio/internal/setup"
)

func main() {
	log := logger.NewLogger()
	c := providers.NewClusterProvider(log, database.GetDB())

	setup.Run(&setup.Deps{
		Log:     log,
		DB:      database.GetDB(),
		Cluster: c,
	})

	// Migrate this to tests
	// buildingId, err := services.ConstructBuilding(models.Building{
	// 	CityId: "c3b81b20-e975-4a2c-ae93-c81bf8e1303d",
	// 	Type:   "barracks",
	// 	Level:  1,
	// 	X:      31,
	// 	Y:      6,
	// })
	// if err != nil {
	// 	panic(err)
	// }
	// time.Sleep(time.Second * 11)
	// log.Printf("Created barracks with id %s", buildingId)

	// err = services.TrainTroops(models.Training{
	// 	BarracksId: buildingId,
	// 	Size:       20,
	// })

	// time.Sleep(time.Second * (constants.TROOP_TRAINING_DURATION + 1))

	// err = services.TrainTroops(models.Training{
	// 	BarracksId: buildingId,
	// 	Size:       10,
	// 	DeployTo:   "164bab00-3fc7-41a8-bf76-22d6bba42f2a",
	// })

	api.Start()
}
