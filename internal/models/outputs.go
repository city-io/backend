package models

type MapTileOutput struct {
	X    int   `json:"x"`
	Y    int   `json:"y"`
	City *City `json:"city"`
}
