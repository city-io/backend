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

// UserToProto converts a domain user to its proto representation. The password
// is never copied across the wire.
func UserToProto(u domain.User) *entityv1.User {
	return &entityv1.User{
		UserId:         ToUserId(u.UserID),
		Email:          u.Email,
		Username:       u.Username,
		Gold:           u.Gold,
		Food:           u.Food,
		FoodIncomeRate: u.FoodIncomeRate,
		FoodUpkeepRate: u.FoodUpkeepRate,
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
		FoodProductionRate: c.FoodProductionRate,
		FoodUpkeep:         c.FoodUpkeep,
		NetFoodFlow:        c.NetFoodFlow,
		Starving:           c.Starving,
	}
	if c.Owner != nil {
		out.Owner = ToUserId(*c.Owner)
	}
	return out
}

// TileToProto builds a proto Tile from raw occupancy data.
func TileToProto(cityID, buildingID *string, x, y int) *servicev1.Tile {
	t := &servicev1.Tile{X: int32(x), Y: int32(y)}
	if cityID != nil {
		t.CityId = ToCityId(*cityID)
	}
	if buildingID != nil {
		t.BuildingId = ToBuildingId(*buildingID)
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

// EntitiesToBag builds an EntityBag from slices of domain entities.
func EntitiesToBag(users []domain.User, cities []domain.City, buildings []domain.Building) *entityv1.EntityBag {
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
	return bag
}
