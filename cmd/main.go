package main

import (
	"cityio/internal/api"
	"cityio/internal/services"
)

func main() {
	services.Init()
	api.Start()
}
