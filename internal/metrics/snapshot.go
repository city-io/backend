package metrics

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"cityio/internal/ports"
)

// SnapshotInterval is how often the snapshot goroutine refreshes
// aggregate game-state gauges.
const SnapshotInterval = 5 * time.Second

// StartSnapshot kicks off a background goroutine that periodically walks the
// persistence layer to populate the aggregate game-state gauges (population
// totals, building counts, etc.). The goroutine exits when shutdownCtx is
// cancelled.
func StartSnapshot(shutdownCtx context.Context, store ports.Store) {
	go func() {
		// Take one snapshot immediately so the gauges aren't 0 at boot.
		snapshot(shutdownCtx, store)
		ticker := time.NewTicker(SnapshotInterval)
		defer ticker.Stop()
		for {
			select {
			case <-shutdownCtx.Done():
				return
			case <-ticker.C:
				snapshot(shutdownCtx, store)
			}
		}
	}()
}

func snapshot(ctx context.Context, store ports.Store) {
	if err := snapshotUsers(ctx, store); err != nil {
		slog.ErrorContext(ctx, "metrics snapshot: users", "error", err)
	}
	if err := snapshotCities(ctx, store); err != nil {
		slog.ErrorContext(ctx, "metrics snapshot: cities", "error", err)
	}
	if err := snapshotBuildings(ctx, store); err != nil {
		slog.ErrorContext(ctx, "metrics snapshot: buildings", "error", err)
	}
}

func snapshotUsers(ctx context.Context, store ports.Store) error {
	users, err := store.GetAllUsers(ctx)
	if err != nil {
		return err
	}
	var foodSum, goldSum int64
	for _, u := range users {
		foodSum += u.Food
		goldSum += u.Gold
	}
	UsersTotal.Set(float64(len(users)))
	UserFoodSum.Set(float64(foodSum))
	UserGoldSum.Set(float64(goldSum))
	return nil
}

func snapshotCities(ctx context.Context, store ports.Store) error {
	cities, err := store.GetAllCities(ctx)
	if err != nil {
		return err
	}
	// Reset the labelled gauge before recounting so disappearing label sets
	// don't linger.
	CitiesTotal.Reset()

	var popSum, capSum, prodSum, upkeepSum float64
	var starving int
	for _, c := range cities {
		owned := "false"
		if c.Owner != nil {
			owned = "true"
		}
		CitiesTotal.WithLabelValues(string(c.Type), owned).Inc()
		if c.Owner == nil {
			continue
		}
		popSum += c.Population
		capSum += c.PopulationCap
		prodSum += float64(c.FoodProductionRate)
		upkeepSum += float64(c.FoodUpkeep)
		if c.Starving {
			starving++
		}
	}
	PopulationSum.Set(popSum)
	PopulationCapSum.Set(capSum)
	CitiesStarving.Set(float64(starving))
	FoodProductionRateSum.Set(prodSum)
	FoodUpkeepRateSum.Set(upkeepSum)
	return nil
}

func snapshotBuildings(ctx context.Context, store ports.Store) error {
	buildings, err := store.GetAllBuildings(ctx)
	if err != nil {
		return err
	}
	BuildingsTotal.Reset()
	for _, b := range buildings {
		BuildingsTotal.WithLabelValues(b.Type, strconv.Itoa(b.Level)).Inc()
	}
	return nil
}
