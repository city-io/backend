package main

import (
	"cityio/internal/api"
	"cityio/internal/services"
)

func main() {
	services.Reset()
	services.Init()
	api.Start()
}
