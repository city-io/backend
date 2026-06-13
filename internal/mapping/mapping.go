// Package mapping converts between pure domain entities and the generated
// protobuf transport types. It is the only place that knows about both, so the
// domain package stays free of transport concerns.
package mapping

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "cityio/internal/gen/cityio/v1"

	"cityio/internal/domain"
)

var cityTypeToProto = map[domain.CityType]pb.CityType{
	domain.CityTypeCity: pb.CityType_CITY_TYPE_CITY,
	domain.CityTypeTown: pb.CityType_CITY_TYPE_TOWN,
}

var cityTypeFromProto = map[pb.CityType]domain.CityType{
	pb.CityType_CITY_TYPE_CITY: domain.CityTypeCity,
	pb.CityType_CITY_TYPE_TOWN: domain.CityTypeTown,
}

var buildingTypeToProto = map[domain.BuildingType]pb.BuildingType{
	domain.BuildingTypeCityCenter: pb.BuildingType_BUILDING_TYPE_CITY_CENTER,
	domain.BuildingTypeTownCenter: pb.BuildingType_BUILDING_TYPE_TOWN_CENTER,
	domain.BuildingTypeBarracks:   pb.BuildingType_BUILDING_TYPE_BARRACKS,
	domain.BuildingTypeHouse:      pb.BuildingType_BUILDING_TYPE_HOUSE,
	domain.BuildingTypeFarm:       pb.BuildingType_BUILDING_TYPE_FARM,
	domain.BuildingTypeMine:       pb.BuildingType_BUILDING_TYPE_MINE,
}

var buildingTypeFromProto = map[pb.BuildingType]domain.BuildingType{
	pb.BuildingType_BUILDING_TYPE_CITY_CENTER: domain.BuildingTypeCityCenter,
	pb.BuildingType_BUILDING_TYPE_TOWN_CENTER: domain.BuildingTypeTownCenter,
	pb.BuildingType_BUILDING_TYPE_BARRACKS:    domain.BuildingTypeBarracks,
	pb.BuildingType_BUILDING_TYPE_HOUSE:       domain.BuildingTypeHouse,
	pb.BuildingType_BUILDING_TYPE_FARM:        domain.BuildingTypeFarm,
	pb.BuildingType_BUILDING_TYPE_MINE:        domain.BuildingTypeMine,
}

// CityTypeToProto maps a domain city type to its proto enum.
func CityTypeToProto(t domain.CityType) pb.CityType {
	return cityTypeToProto[t]
}

// CityTypeFromProto maps a proto city type enum to its domain value.
func CityTypeFromProto(t pb.CityType) domain.CityType {
	return cityTypeFromProto[t]
}

// BuildingTypeToProto maps a domain building type to its proto enum.
func BuildingTypeToProto(t domain.BuildingType) pb.BuildingType {
	return buildingTypeToProto[t]
}

// BuildingTypeFromProto maps a proto building type enum to its domain value.
func BuildingTypeFromProto(t pb.BuildingType) domain.BuildingType {
	return buildingTypeFromProto[t]
}

// UserToProto converts a domain user to its proto representation. The password
// is never copied across the wire.
func UserToProto(u domain.User) *pb.User {
	return &pb.User{
		UserId:   u.UserID,
		Email:    u.Email,
		Username: u.Username,
		Gold:     u.Gold,
		Food:     u.Food,
	}
}

// CityToProto converts a domain city to its proto representation.
func CityToProto(c domain.City) *pb.City {
	return &pb.City{
		CityId:        c.CityID,
		Type:          CityTypeToProto(c.Type),
		Owner:         c.Owner,
		Name:          c.Name,
		Population:    c.Population,
		PopulationCap: c.PopulationCap,
		Start:         &pb.Coordinates{X: int32(c.StartX), Y: int32(c.StartY)},
		Size:          int32(c.Size),
	}
}

// TileToProto builds a proto Tile from raw occupancy data.
func TileToProto(cityID, buildingID *string, x, y int) *pb.Tile {
	return &pb.Tile{X: int32(x), Y: int32(y), CityId: cityID, BuildingId: buildingID}
}

// BuildingToProto converts a domain building to its proto representation.
func BuildingToProto(b domain.Building) *pb.Building {
	out := &pb.Building{
		BuildingId:  b.BuildingID,
		CityId:      b.CityID,
		Type:        BuildingTypeToProto(b.BuildingType()),
		Level:       int32(b.Level),
		TargetLevel: int32(b.TargetLevel),
		Coords:      &pb.Coordinates{X: int32(b.X), Y: int32(b.Y)},
	}
	if b.ConstructionStart.Time != nil {
		out.ConstructionStart = timestamppb.New(*b.ConstructionStart.Time)
	}
	if b.ConstructionEnd.Time != nil {
		out.ConstructionEnd = timestamppb.New(*b.ConstructionEnd.Time)
	}
	return out
}
