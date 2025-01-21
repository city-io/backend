package services

import (
	"cityio/internal/actors"
	"cityio/internal/database"
)

var system = actors.GetSystem()
var db = database.GetDb()
