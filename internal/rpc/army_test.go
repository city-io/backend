package rpc

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	"cityio/internal/domain"
	servicev1 "cityio/internal/gen/cityio/service/v1"
	"cityio/internal/messages"
)

type routeTestWorld struct {
	grid domain.TerrainGrid
}

func (w routeTestWorld) Terrain() domain.TerrainGrid { return w.grid }
func (w routeTestWorld) TerrainAt(x, y int) (domain.TerrainType, bool) {
	return w.grid.At(x, y)
}
func (routeTestWorld) ReserveCity(int) (domain.Coordinates, error) { return domain.Coordinates{}, nil }

func TestMoveArmyRequiresDestination(t *testing.T) {
	handler := &armyHandler{}
	_, err := handler.MoveArmy(context.Background(), connect.NewRequest(&servicev1.MoveArmyRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want %v", connect.CodeOf(err), connect.CodeInvalidArgument)
	}
}

func TestTrainingErrorRejectsInvalidTroopType(t *testing.T) {
	err := trainingError(&messages.InvalidTroopTypeError{})
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want %v", connect.CodeOf(err), connect.CodeInvalidArgument)
	}
}

func TestArmyOrderProjectsClosestKnownLand(t *testing.T) {
	destination := 2
	orderID := "order"
	army := domain.Army{
		ArmyID: "army", Owner: "owner", Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 1},
		DestX: &destination, DestY: new(int), OrderID: &orderID, OrderKind: domain.ArmyOrderMove,
	}
	server := &Server{world: routeTestWorld{grid: domain.TerrainGrid{
		Width: 3, Height: 1,
		Tiles: []domain.TerrainType{domain.TerrainTypeGrassland, domain.TerrainTypeGrassland, domain.TerrainTypeWater},
	}}}
	explored := map[domain.Coordinates]struct{}{{X: 0}: {}, {X: 1}: {}, {X: 2}: {}}

	order := server.projectOwnedArmyOrder(army, explored)

	if order == nil || order.GetMove().GetDestination().GetX() != 2 || len(order.GetRemainingRoute()) != 1 {
		t.Fatalf("order projection = %+v", order)
	}
	last := order.GetRemainingRoute()[0].GetCoords()
	if last.GetX() != 1 || last.GetY() != 0 {
		t.Fatalf("route endpoint = (%d,%d), want closest land (1,0)", last.GetX(), last.GetY())
	}
	if got := order.GetEstimatedRemainingDuration().AsDuration(); got != 1250*time.Millisecond {
		t.Fatalf("estimated duration = %s, want 1.25s", got)
	}
}

func TestArmyOrderProjectsActorsRemainingPath(t *testing.T) {
	destination := 2
	orderID := "order"
	army := domain.Army{
		ArmyID: "army", Owner: "owner", Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 1},
		DestX: &destination, DestY: new(int), OrderID: &orderID, OrderKind: domain.ArmyOrderMove,
		RemainingPath: []domain.Coordinates{{X: 1, Y: 1}, {X: 2, Y: 0}},
	}
	server := &Server{world: routeTestWorld{grid: domain.TerrainGrid{
		Width: 3, Height: 2,
		Tiles: []domain.TerrainType{
			domain.TerrainTypeGrassland, domain.TerrainTypeGrassland, domain.TerrainTypeGrassland,
			domain.TerrainTypeGrassland, domain.TerrainTypeGrassland, domain.TerrainTypeGrassland,
		},
	}}}
	explored := map[domain.Coordinates]struct{}{}
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			explored[domain.Coordinates{X: x, Y: y}] = struct{}{}
		}
	}

	order := server.projectOwnedArmyOrder(army, explored)
	if order == nil || len(order.GetRemainingRoute()) != 2 {
		t.Fatalf("order projection = %+v", order)
	}
	first := order.GetRemainingRoute()[0].GetCoords()
	if first.GetX() != 1 || first.GetY() != 1 {
		t.Fatalf("projected path starts at (%d,%d), want actor path (1,1)", first.GetX(), first.GetY())
	}
}

func TestArmyOrderHidesUnexploredRouteGeometry(t *testing.T) {
	destination := 3
	orderID := "order"
	army := domain.Army{
		ArmyID: "army", Owner: "owner", Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 1},
		DestX: &destination, DestY: new(int), OrderID: &orderID, OrderKind: domain.ArmyOrderMove,
		RemainingPath: []domain.Coordinates{{X: 1}, {X: 1, Y: 1}, {X: 2, Y: 1}, {X: 3}},
	}
	server := &Server{world: routeTestWorld{grid: domain.TerrainGrid{
		Width: 4, Height: 2, Tiles: make([]domain.TerrainType, 8),
	}}}
	explored := map[domain.Coordinates]struct{}{{X: 0}: {}, {X: 1}: {}}

	order := server.projectOwnedArmyOrder(army, explored)
	if order == nil || len(order.GetRemainingRoute()) != 2 {
		t.Fatalf("disclosed route = %+v, want known prefix and endpoint", order)
	}
	first := order.GetRemainingRoute()[0].GetCoords()
	last := order.GetRemainingRoute()[1].GetCoords()
	if first.GetX() != 1 || first.GetY() != 0 || last.GetX() != 3 || last.GetY() != 0 {
		t.Fatalf("disclosed route = (%d,%d) -> (%d,%d)", first.GetX(), first.GetY(), last.GetX(), last.GetY())
	}
}
