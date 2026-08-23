package messages

import (
	"fmt"

	"cityio/internal/constants"
	"cityio/internal/domain"
)

type CreateCityMessage struct {
	City    domain.City
	Restore bool
}

type UpdateCityOwnerMessage struct {
	Owner *string
}

type CaptureCityMessage struct{ Owner string }

type UpdateCityPolicyMessage struct {
	MilitiaTarget  float64
	TaxRatePercent int
}

type ApplyMilitiaCasualtiesMessage struct{ Count int64 }
type ApplyMilitiaCasualtiesResponseMessage struct {
	Survived bool
}

type BeginMilitiaBattleMessage struct{ BattleID string }
type BeginMilitiaBattleResponseMessage struct {
	BattleID string
	Count    int64
}
type EndMilitiaBattleMessage struct{ BattleID string }

// SetBuildingPopulationMessage reports a building's absolute contribution to its
// city's population cap. Keyed by building so resends are idempotent and the cap
// can be fully rebuilt from its buildings.
type SetBuildingPopulationMessage struct {
	BuildingID string
	Population float64
}

// SetBuildingFoodProductionMessage reports a building's current hourly food
// capacity. Keying by building makes restore-time and periodic resends
// idempotent, just like population-cap contributions.
type SetBuildingFoodProductionMessage struct {
	BuildingID    string
	AmountPerHour int64
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

type CreditOwnerGoldMessage struct {
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

type InvalidCityPolicyError struct{}

func (*InvalidCityPolicyError) Error() string {
	return fmt.Sprintf(
		"militia target must be a whole resident within %d-%d%% of housing capacity and tax rate must be 0-%d%%",
		constants.MinMilitiaPercent,
		constants.MaxMilitiaPercent,
		constants.MaxTaxRatePercent,
	)
}

type CityPolicyLockedError struct{}

func (*CityPolicyLockedError) Error() string {
	return "city policy cannot change while its militia is in battle"
}

func (e *CityNotFoundError) Error() string {
	return fmt.Sprintf("City not found: %s", e.CityId)
}
