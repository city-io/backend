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
	_, ctrls := providers.NewRuntime(log, database.GetDB())

	setup.Run(&setup.Deps{
		Log:         log,
		DB:          database.GetDB(),
		Controllers: ctrls,
	})

	api.Start()
}
