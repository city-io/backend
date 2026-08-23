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

func TestArmyMarchProjectsClosestKnownLandWithoutRedundantBoolean(t *testing.T) {
	destination := 2
	marchID := "march"
	army := domain.Army{
		ArmyID: "army", Owner: "owner", Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 1},
		DestX: &destination, DestY: new(int), MarchID: &marchID,
	}
	server := &Server{world: routeTestWorld{grid: domain.TerrainGrid{
		Width: 3, Height: 1,
		Tiles: []domain.TerrainType{domain.TerrainTypeGrassland, domain.TerrainTypeGrassland, domain.TerrainTypeWater},
	}}}
	explored := map[domain.Coordinates]struct{}{{X: 0}: {}, {X: 1}: {}, {X: 2}: {}}

	march := server.projectOwnedArmyMarch(army, explored)

	if march == nil || march.GetDestination().GetX() != 2 || len(march.GetRemainingRoute()) != 1 {
		t.Fatalf("march projection = %+v", march)
	}
	last := march.GetRemainingRoute()[0].GetCoords()
	if last.GetX() != 1 || last.GetY() != 0 {
		t.Fatalf("route endpoint = (%d,%d), want closest land (1,0)", last.GetX(), last.GetY())
	}
	if got := march.GetEstimatedRemainingDuration().AsDuration(); got != 1250*time.Millisecond {
		t.Fatalf("estimated duration = %s, want 1.25s", got)
	}
}

func TestArmyMarchProjectsActorsRemainingPath(t *testing.T) {
	destination := 2
	marchID := "march"
	army := domain.Army{
		ArmyID: "army", Owner: "owner", Troops: map[domain.TroopType]int64{domain.TroopTypeSoldier: 1},
		DestX: &destination, DestY: new(int), MarchID: &marchID,
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

	march := server.projectOwnedArmyMarch(army, explored)
	if march == nil || len(march.GetRemainingRoute()) != 2 {
		t.Fatalf("march projection = %+v", march)
	}
	first := march.GetRemainingRoute()[0].GetCoords()
	if first.GetX() != 1 || first.GetY() != 1 {
		t.Fatalf("projected path starts at (%d,%d), want actor path (1,1)", first.GetX(), first.GetY())
	}
}
