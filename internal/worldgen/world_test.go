package worldgen

import (
	"math"
	"reflect"
	"testing"

	"cityio/internal/domain"
)

func testConfig(seed int64) Config {
	return Config{
		Seed:         seed,
		Width:        75,
		Height:       75,
		CapitalSize:  5,
		CapitalSites: 12,
		TownTarget:   64,
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	first, err := Generate(testConfig(42))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate(testConfig(42))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(first.terrain, second.terrain) {
		t.Fatal("terrain differs for the same seed")
	}
	if !reflect.DeepEqual(first.capitalSite, second.capitalSite) {
		t.Fatal("capital sites differ for the same seed")
	}
	if !reflect.DeepEqual(first.towns, second.towns) {
		t.Fatal("town plans differ for the same seed")
	}
}

func TestGenerateVariesBySeed(t *testing.T) {
	first, err := Generate(testConfig(1))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate(testConfig(2))
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(first.terrain, second.terrain) {
		t.Fatal("terrain is identical for different seeds")
	}
}

func TestGeneratedSettlementsAreValid(t *testing.T) {
	for seed := int64(0); seed < 25; seed++ {
		world, err := Generate(testConfig(seed))
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if len(world.towns) != world.config.TownTarget {
			t.Fatalf("seed %d: towns = %d, want %d", seed, len(world.towns), world.config.TownTarget)
		}

		occupied := make([]bool, world.config.Width*world.config.Height)
		for _, site := range world.capitalSite {
			if !canPlace(world.terrain, occupied, site.X, site.Y, world.config.CapitalSize, settlementGap) {
				t.Fatalf("seed %d: invalid capital site at %d,%d", seed, site.X, site.Y)
			}
			markOccupied(occupied, world.config.Width, site.X, site.Y, world.config.CapitalSize)
		}
		for _, town := range world.towns {
			if !canPlace(world.terrain, occupied, town.X, town.Y, town.Size, settlementGap) {
				t.Fatalf("seed %d: invalid town %s at %d,%d", seed, town.Name, town.X, town.Y)
			}
			markOccupied(occupied, world.config.Width, town.X, town.Y, town.Size)
		}

		for i, first := range world.capitalSite {
			for _, second := range world.capitalSite[i+1:] {
				dx := first.X - second.X
				dy := first.Y - second.Y
				if math.Hypot(float64(dx), float64(dy)) < capitalMinSeparation {
					t.Fatalf("seed %d: capital sites are too close", seed)
				}
			}
		}
	}
}

func TestTerrainHasNoTinyInteriorRegions(t *testing.T) {
	world, err := Generate(testConfig(99))
	if err != nil {
		t.Fatal(err)
	}
	visited := make([]bool, len(world.terrain.Tiles))
	for start, terrain := range world.terrain.Tiles {
		if visited[start] {
			continue
		}
		region := collectRegion(world.terrain.Tiles, visited, world.terrain.Width, world.terrain.Height, start)
		if terrain != domain.TerrainTypeWater && len(region) < minTerrainRegionSize {
			t.Fatalf("%s region has only %d tiles", terrain, len(region))
		}
	}
}

func TestTerrainIncludesEveryType(t *testing.T) {
	wanted := []domain.TerrainType{
		domain.TerrainTypeGrassland,
		domain.TerrainTypePlains,
		domain.TerrainTypeForest,
		domain.TerrainTypeHills,
		domain.TerrainTypeMountains,
		domain.TerrainTypeDesert,
		domain.TerrainTypeMarsh,
		domain.TerrainTypeWater,
	}
	for seed := int64(0); seed < 10; seed++ {
		world, err := Generate(testConfig(seed))
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		counts := make(map[domain.TerrainType]int)
		for _, terrain := range world.terrain.Tiles {
			counts[terrain]++
		}
		largest := make(map[domain.TerrainType]int)
		visited := make([]bool, len(world.terrain.Tiles))
		for start, terrain := range world.terrain.Tiles {
			if visited[start] {
				continue
			}
			region := collectRegion(world.terrain.Tiles, visited, world.terrain.Width, world.terrain.Height, start)
			largest[terrain] = max(largest[terrain], len(region))
		}
		for _, terrain := range wanted {
			if counts[terrain] == 0 {
				t.Fatalf("seed %d: missing %s", seed, terrain)
			}
			if largest[terrain] < 8 {
				t.Fatalf("seed %d: largest %s region has only %d tiles", seed, terrain, largest[terrain])
			}
		}
	}
}

func TestReserveCityUsesGeneratedCapitalSites(t *testing.T) {
	config := testConfig(123)
	world, err := Generate(config)
	if err != nil {
		t.Fatal(err)
	}

	for i, want := range world.capitalSite {
		got, err := world.ReserveCity(config.CapitalSize)
		if err != nil {
			t.Fatalf("reservation %d: %v", i, err)
		}
		if got != want {
			t.Fatalf("reservation %d = %+v, want %+v", i, got, want)
		}
	}
}
