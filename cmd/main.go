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
	db := database.NewDB(log)
	_, ctrls := providers.NewRuntime(log, db)

	setup.Run(&setup.Deps{
		Log:         log,
		DB:          db,
		Controllers: ctrls,
	})

	api.Start()
}
