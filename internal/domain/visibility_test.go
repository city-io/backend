package domain

import "testing"

func TestVisionIncludesArmyRadius(t *testing.T) {
	vision := Vision{Armies: []Army{{ArmyID: "scout", X: 10, Y: 10}}}

	for _, point := range []struct{ x, y int }{{7, 7}, {10, 13}, {13, 10}} {
		if !vision.PointVisible(point.x, point.y, 3) {
			t.Errorf("point (%d,%d) should be visible", point.x, point.y)
		}
	}
	if vision.PointVisible(14, 10, 3) {
		t.Fatal("point outside army radius should not be visible")
	}
}

func TestVisionIncludesIndependentPointRadius(t *testing.T) {
	vision := Vision{Points: []VisionPoint{{X: 10, Y: 10, Radius: 5}}}
	if !vision.PointVisible(15, 15, 3) {
		t.Fatal("point inside structure-specific radius should be visible")
	}
	if vision.PointVisible(16, 10, 3) {
		t.Fatal("point outside structure-specific radius should not be visible")
	}
}

func TestVisionArmyRevealsIntersectingCityAndEntities(t *testing.T) {
	vision := Vision{Armies: []Army{{ArmyID: "scout", X: 5, Y: 5}}}
	cities := []City{
		{CityID: "visible", StartX: 8, StartY: 8, Size: 2},
		{CityID: "hidden", StartX: 9, StartY: 9, Size: 2},
	}
	buildings := []Building{
		{BuildingID: "visible", X: 8, Y: 5},
		{BuildingID: "hidden", X: 9, Y: 5},
	}
	armies := []Army{
		{ArmyID: "visible", X: 2, Y: 2},
		{ArmyID: "hidden", X: 1, Y: 1},
	}

	if got := vision.FilterCities(cities, 3); len(got) != 1 || got[0].CityID != "visible" {
		t.Fatalf("visible cities = %+v, want visible", got)
	}
	if got := vision.FilterBuildings(buildings, 3); len(got) != 1 || got[0].BuildingID != "visible" {
		t.Fatalf("visible buildings = %+v, want visible", got)
	}
	if got := vision.FilterArmies(armies, 3); len(got) != 1 || got[0].ArmyID != "visible" {
		t.Fatalf("visible armies = %+v, want visible", got)
	}
}

func TestVisionCombinesCityAndArmySources(t *testing.T) {
	vision := Vision{
		Cities: []City{{CityID: "capital", StartX: 1, StartY: 1, Size: 2}},
		Armies: []Army{{ArmyID: "scout", X: 20, Y: 20}},
	}

	if !vision.PointVisible(0, 0, 1) {
		t.Fatal("city vision source should remain active")
	}
	if !vision.PointVisible(21, 21, 1) {
		t.Fatal("army vision source should be active")
	}
}

func TestVisibleCoordinatesAreClampedAndDeduplicated(t *testing.T) {
	vision := Vision{
		Cities: []City{{StartX: 0, StartY: 0, Size: 2}},
		Armies: []Army{{X: 2, Y: 2}},
	}
	coords := vision.VisibleCoordinates(4, 4, 1)
	seen := make(map[Coordinates]struct{}, len(coords))
	for _, coords := range coords {
		if coords.X < 0 || coords.Y < 0 || coords.X >= 4 || coords.Y >= 4 {
			t.Fatalf("out-of-bounds coordinate: %+v", coords)
		}
		if _, exists := seen[coords]; exists {
			t.Fatalf("duplicate coordinate: %+v", coords)
		}
		seen[coords] = struct{}{}
	}
	if len(coords) != 14 {
		t.Fatalf("visible coordinates = %d, want 14", len(coords))
	}
}
