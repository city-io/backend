package controllers

import (
	"cityio/internal/logger"
	"cityio/internal/ports"
)

type TileController struct {
	cluster ports.ClusterProvider
	log     logger.Logger
}

func NewTileController(cl ports.ClusterProvider, l logger.Logger) *TileController {
	return &TileController{
		cluster: cl,
		log:     l,
	}
}
