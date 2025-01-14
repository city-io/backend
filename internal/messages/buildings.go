package messages

import "fmt"

type BuildingTypeNotFoundError struct {
	BuildingType string
}

func (e *BuildingTypeNotFoundError) Error() string {
	return fmt.Sprintf("Building type not found: %s", e.BuildingType)
}
