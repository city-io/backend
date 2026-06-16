package messages

import (
	"fmt"

	"cityio/internal/domain"
)

type CreateCityMessage struct {
	City    domain.City
	Restore bool
}

type UpdateCityOwnerMessage struct {
	Owner *string
}

// SetBuildingPopulationMessage reports a building's absolute contribution to its
// city's population cap. Keyed by building so resends are idempotent and the cap
// can be fully rebuilt from its buildings.
type SetBuildingPopulationMessage struct {
	BuildingID string
	Population float64
}

// CreditProductionMessage routes a building's produced resources to its city,
// which forwards them to the city's owner (if any). The city owns the owner, so
// buildings never cache it.
type CreditProductionMessage struct {
	Gold int64
	Food int64
}

// DeductOwnerGoldMessage asks a city to deduct gold from its owner (e.g. for a
// building upgrade), relaying the owner's Ack or InsufficientGoldError.
type DeductOwnerGoldMessage struct {
	Amount int64
}

type BuildingDestroyedMessage struct {
	BuildingID string
}

type GetCityMessage struct{}
type GetCityResponseMessage struct {
	City domain.City
}

type DeleteCityMessage struct {
	CityID string
}

// Errors
type CityNotFoundError struct {
	CityId string
}

func (e *CityNotFoundError) Error() string {
	return fmt.Sprintf("City not found: %s", e.CityId)
}
