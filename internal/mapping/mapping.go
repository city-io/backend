// Package mapping converts between pure domain entities and the generated
// protobuf transport types. It is the only place that knows about both, so the
// domain package stays free of transport concerns.
package mapping

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	entityv1 "cityio/internal/gen/cityio/entity/v1"
	servicev1 "cityio/internal/gen/cityio/service/v1"

	"cityio/internal/domain"
)

var cityTypeToProto = map[domain.CityType]entityv1.CityType{
	domain.CityTypeCity: entityv1.CityType_CITY_TYPE_CITY,
	domain.CityTypeTown: entityv1.CityType_CITY_TYPE_TOWN,
}

var cityTypeFromProto = map[entityv1.CityType]domain.CityType{
	entityv1.CityType_CITY_TYPE_CITY: domain.CityTypeCity,
	entityv1.CityType_CITY_TYPE_TOWN: domain.CityTypeTown,
}

var buildingTypeToProto = map[domain.BuildingType]entityv1.BuildingType{
	domain.BuildingTypeCityCenter: entityv1.BuildingType_BUILDING_TYPE_CITY_CENTER,
	domain.BuildingTypeTownCenter: entityv1.BuildingType_BUILDING_TYPE_TOWN_CENTER,
	domain.BuildingTypeBarracks:   entityv1.BuildingType_BUILDING_TYPE_BARRACKS,
	domain.BuildingTypeHouse:      entityv1.BuildingType_BUILDING_TYPE_HOUSE,
	domain.BuildingTypeFarm:       entityv1.BuildingType_BUILDING_TYPE_FARM,
	domain.BuildingTypeMine:       entityv1.BuildingType_BUILDING_TYPE_MINE,
}

var buildingTypeFromProto = map[entityv1.BuildingType]domain.BuildingType{
	entityv1.BuildingType_BUILDING_TYPE_CITY_CENTER: domain.BuildingTypeCityCenter,
	entityv1.BuildingType_BUILDING_TYPE_TOWN_CENTER: domain.BuildingTypeTownCenter,
	entityv1.BuildingType_BUILDING_TYPE_BARRACKS:    domain.BuildingTypeBarracks,
	entityv1.BuildingType_BUILDING_TYPE_HOUSE:       domain.BuildingTypeHouse,
	entityv1.BuildingType_BUILDING_TYPE_FARM:        domain.BuildingTypeFarm,
	entityv1.BuildingType_BUILDING_TYPE_MINE:        domain.BuildingTypeMine,
}

var troopTypeToProto = map[domain.TroopType]entityv1.TroopType{
	domain.TroopTypeSoldier:   entityv1.TroopType_TROOP_TYPE_SOLDIER,
	domain.TroopTypeArcher:    entityv1.TroopType_TROOP_TYPE_ARCHER,
	domain.TroopTypeCavalry:   entityv1.TroopType_TROOP_TYPE_CAVALRY,
	domain.TroopTypeArtillery: entityv1.TroopType_TROOP_TYPE_ARTILLERY,
}

var troopTypeFromProto = map[entityv1.TroopType]domain.TroopType{
	entityv1.TroopType_TROOP_TYPE_SOLDIER:   domain.TroopTypeSoldier,
	entityv1.TroopType_TROOP_TYPE_ARCHER:    domain.TroopTypeArcher,
	entityv1.TroopType_TROOP_TYPE_CAVALRY:   domain.TroopTypeCavalry,
	entityv1.TroopType_TROOP_TYPE_ARTILLERY: domain.TroopTypeArtillery,
}

// ToUserId wraps a raw string into a typed proto ID.
func ToUserId(id string) *entityv1.UserId {
	return &entityv1.UserId{Value: id}
}

// ToCityId wraps a raw string into a typed proto ID.
func ToCityId(id string) *entityv1.CityId {
	return &entityv1.CityId{Value: id}
}

// ToBuildingId wraps a raw string into a typed proto ID.
func ToBuildingId(id string) *entityv1.BuildingId {
	return &entityv1.BuildingId{Value: id}
}

// ToArmyId wraps a raw string into a typed proto ID.
func ToArmyId(id string) *entityv1.ArmyId {
	return &entityv1.ArmyId{Value: id}
}

// TroopTypeToProto maps a domain troop type to its proto enum.
func TroopTypeToProto(t domain.TroopType) entityv1.TroopType {
	return troopTypeToProto[t]
}

// TroopTypeFromProto maps a proto troop type enum to its domain value.
func TroopTypeFromProto(t entityv1.TroopType) domain.TroopType {
	return troopTypeFromProto[t]
}

// CityTypeToProto maps a domain city type to its proto enum.
func CityTypeToProto(t domain.CityType) entityv1.CityType {
	return cityTypeToProto[t]
}

// CityTypeFromProto maps a proto city type enum to its domain value.
func CityTypeFromProto(t entityv1.CityType) domain.CityType {
	return cityTypeFromProto[t]
}

// BuildingTypeToProto maps a domain building type to its proto enum.
func BuildingTypeToProto(t domain.BuildingType) entityv1.BuildingType {
	return buildingTypeToProto[t]
}

// BuildingTypeFromProto maps a proto building type enum to its domain value.
func BuildingTypeFromProto(t entityv1.BuildingType) domain.BuildingType {
	return buildingTypeFromProto[t]
}

// RatePerHour wraps a per-hour amount as a Rate proto with scale=3600.
func RatePerHour(perHour int64) *entityv1.Rate {
	return &entityv1.Rate{Value: perHour, Scale: 3600}
}

// UserToProto converts a domain user to its proto representation. The password
// is never copied across the wire.
func UserToProto(u domain.User) *entityv1.User {
	return &entityv1.User{
		UserId:     ToUserId(u.UserID),
		Email:      u.Email,
		Username:   u.Username,
		Gold:       u.Gold,
		Food:       u.Food,
		FoodIncome: RatePerHour(u.FoodIncomeRate),
		FoodUpkeep: RatePerHour(u.FoodUpkeepRate),
	}
}

// CityToProto converts a domain city to its proto representation.
func CityToProto(c domain.City) *entityv1.City {
	out := &entityv1.City{
		CityId:             ToCityId(c.CityID),
		Type:               CityTypeToProto(c.Type),
		Name:               c.Name,
		Population:         c.Population,
		PopulationCap:      c.PopulationCap,
		Start:              &entityv1.Coordinates{X: int32(c.StartX), Y: int32(c.StartY)},
		Size:               int32(c.Size),
		FoodProduction:     RatePerHour(c.FoodProductionRate),
		FoodUpkeep:         RatePerHour(c.FoodUpkeep),
		NetFoodFlow:        RatePerHour(c.NetFoodFlow),
		Starving:           c.Starving,
		PopulationGrowth:   RatePerHour(c.PopulationGrowthRate),
		MilitaryPopulation: c.MilitaryPopulation,
	}
	if c.Owner != nil {
		out.Owner = ToUserId(*c.Owner)
	}
	return out
}

// HidePrivateCityFields blanks the production/upkeep rate fields on a city
// proto. Call this when the viewer is not the city's owner: only the owner
// gets to see economic intel (food_production, food_upkeep, net_food_flow).
// Public fields (identity, location, population, population_cap, starving)
// stay untouched. See the visibility note on the City proto.
func HidePrivateCityFields(c *entityv1.City) {
	c.FoodProduction = nil
	c.FoodUpkeep = nil
	c.NetFoodFlow = nil
}

// TileToProto builds a proto Tile from raw occupancy data.
func TileToProto(cityID, buildingID *string, armyIDs []string, x, y int) *servicev1.Tile {
	t := &servicev1.Tile{X: int32(x), Y: int32(y)}
	if cityID != nil {
		t.CityId = ToCityId(*cityID)
	}
	if buildingID != nil {
		t.BuildingId = ToBuildingId(*buildingID)
	}
	for _, id := range armyIDs {
		t.ArmyIds = append(t.ArmyIds, ToArmyId(id))
	}
	return t
}

// BuildingToProto converts a domain building to its proto representation.
func BuildingToProto(b domain.Building) *entityv1.Building {
	out := &entityv1.Building{
		BuildingId:  ToBuildingId(b.BuildingID),
		CityId:      ToCityId(b.CityID),
		Type:        BuildingTypeToProto(b.BuildingType()),
		Level:       int32(b.Level),
		TargetLevel: int32(b.TargetLevel),
		Coords:      &entityv1.Coordinates{X: int32(b.X), Y: int32(b.Y)},
	}
	if b.ConstructionStart.Time != nil {
		out.ConstructionStart = timestamppb.New(*b.ConstructionStart.Time)
	}
	if b.ConstructionEnd.Time != nil {
		out.ConstructionEnd = timestamppb.New(*b.ConstructionEnd.Time)
	}
	return out
}

// ArmyToProto converts a domain army to its proto representation.
func ArmyToProto(a domain.Army) *entityv1.Army {
	out := &entityv1.Army{
		ArmyId: ToArmyId(a.ArmyID),
		Owner:  ToUserId(a.Owner),
		Coords: &entityv1.Coordinates{X: int32(a.X), Y: int32(a.Y)},
	}
	for troopType, count := range a.Troops {
		out.Troops = append(out.Troops, &entityv1.TroopStack{
			Type:  TroopTypeToProto(troopType),
			Count: int32(count),
		})
	}
	if a.DestX != nil && a.DestY != nil {
		out.Destination = &entityv1.Coordinates{X: int32(*a.DestX), Y: int32(*a.DestY)}
	}
	return out
}

// EntitiesToBag builds an EntityBag from slices of domain entities.
func EntitiesToBag(users []domain.User, cities []domain.City, buildings []domain.Building, armies []domain.Army) *entityv1.EntityBag {
	bag := &entityv1.EntityBag{}
	for _, u := range users {
		bag.Users = append(bag.Users, UserToProto(u))
	}
	for _, c := range cities {
		bag.Cities = append(bag.Cities, CityToProto(c))
	}
	for _, b := range buildings {
		bag.Buildings = append(bag.Buildings, BuildingToProto(b))
	}
	for _, a := range armies {
		bag.Armies = append(bag.Armies, ArmyToProto(a))
	}
	return bag
}
