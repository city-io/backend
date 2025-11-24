package utils

import "strconv"

func GetTileIndex(x, y int) string {
	return strconv.Itoa(x) + "," + strconv.Itoa(y)
}
