package main

import (
	"cityio/internal/api"
	"cityio/internal/cluster"
	"cityio/internal/database"
	"cityio/internal/logger"
	"cityio/internal/setup"
)

func main() {
	log := logger.NewLogger()
	db := database.NewDB(log)
	_, ctrls := cluster.NewRuntime(log, db)

	setup.Run(&setup.Deps{
		Log:         log,
		DB:          db,
		Controllers: ctrls,
	})

	api.Start()
}
