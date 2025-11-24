// Package controllers initializes the controllers for handling various application functionalities.
package controllers

import (
	"cityio/internal/ports"
)

type controllers struct {
	user ports.UserController
	city ports.CityController
	tile ports.TileController
}

func (c *controllers) User() ports.UserController { return c.user }
func (c *controllers) City() ports.CityController { return c.city }
func (c *controllers) Tile() ports.TileController { return c.tile }

func NewControllers(cp ports.ClusterProvider, l ports.Logger) ports.Controllers {
	return &controllers{
		user: NewUserController(cp, l),
		city: NewCityController(cp, l),
		tile: NewTileController(cp, l),
	}
}
