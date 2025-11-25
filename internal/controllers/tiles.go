package controllers

import (
	"cityio/internal/ports"
)

type tileController struct {
	cluster ports.ClusterProvider
	log     ports.Logger
}

func NewTileController(cl ports.ClusterProvider, l ports.Logger) ports.TileController {
	return &tileController{
		cluster: cl,
		log:     l,
	}
}
