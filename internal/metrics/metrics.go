// Package metrics centralises Prometheus collectors for the game backend.
// Collectors are package-level vars wired into the default registry via
// promauto; hot-path code calls them directly. The /metrics endpoint and the
// periodic state snapshot live in sibling files.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "cityio"

// --- RPC surface ------------------------------------------------------------

var (
	// RPCRequestsTotal counts every RPC call by service, method, and Connect
	// status code.
	RPCRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "rpc",
		Name:      "requests_total",
		Help:      "Total RPC requests handled, labelled by service, method, and Connect code.",
	}, []string{"service", "method", "code"})

	// RPCDurationSeconds measures handler latency, labelled by service+method
	// (without code, since code is a small set and adding it as a label
	// explodes the cardinality on the histogram buckets).
	RPCDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "rpc",
		Name:      "duration_seconds",
		Help:      "Handler duration.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"service", "method"})

	// RPCInFlight is the count of handlers currently executing, useful for
	// spotting back-pressure or stuck handlers.
	RPCInFlight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "rpc",
		Name:      "in_flight",
		Help:      "Handlers currently executing.",
	}, []string{"service", "method"})
)

// --- Stream / pub-sub -------------------------------------------------------

var (
	// StreamSubscribers tracks the number of open StreamState subscriptions.
	StreamSubscribers = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "stream",
		Name:      "subscribers",
		Help:      "Active StreamState subscriber count.",
	})

	// StreamPublishesTotal counts pub-sub publishes by what kind of update they
	// carry. A single StateUpdate can mention multiple types; each contributes.
	StreamPublishesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "stream",
		Name:      "publishes_total",
		Help:      "Stream publishes by update type.",
	}, []string{"type"})

	// StreamBufferDropsTotal counts the times a subscriber's buffered channel
	// was full and the publisher had to drop a value to make room.
	StreamBufferDropsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "stream",
		Name:      "buffer_drops_total",
		Help:      "Stream updates dropped because the subscriber's buffer was full.",
	})
)

// --- Persistence ------------------------------------------------------------

var (
	// PersistenceBufferSize is the count of pending entities waiting for the
	// next flush, per kind.
	PersistenceBufferSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "persistence",
		Name:      "buffer_size",
		Help:      "Entities waiting in the flush buffer.",
	}, []string{"kind"})

	// PersistenceFlushDurationSeconds measures end-to-end flush time per kind.
	PersistenceFlushDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "persistence",
		Name:      "flush_duration_seconds",
		Help:      "Time to flush a kind's buffer to the database.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"kind"})

	// PersistenceFlushRowsWritten measures how many rows each flush wrote.
	PersistenceFlushRowsWritten = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "persistence",
		Name:      "flush_rows_written",
		Help:      "Rows written per flush.",
		Buckets:   []float64{0, 1, 10, 100, 500, 1000, 5000, 10000},
	}, []string{"kind"})

	// PersistenceFlushErrorsTotal counts failed flush attempts.
	PersistenceFlushErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "persistence",
		Name:      "flush_errors_total",
		Help:      "Failed flush attempts.",
	}, []string{"kind"})
)

// --- Game state aggregates (set by the snapshot loop) -----------------------

var (
	// UsersTotal is the total number of registered users.
	UsersTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "users_total",
		Help:      "Registered users.",
	})

	// CitiesTotal is cities by type (city / town) and ownership (owned / unowned).
	CitiesTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "cities_total",
		Help:      "Cities by type and ownership.",
	}, []string{"type", "owned"})

	// BuildingsTotal is buildings by type and level.
	BuildingsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "buildings_total",
		Help:      "Buildings by type and level.",
	}, []string{"type", "level"})

	// PopulationSum is the total population across all owned cities.
	PopulationSum = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "population_sum",
		Help:      "Total population across all owned cities.",
	})

	// PopulationCapSum is the total population cap across all owned cities.
	PopulationCapSum = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "population_cap_sum",
		Help:      "Total population cap across all owned cities.",
	})

	// CitiesStarving is the number of owned cities currently flagged starving.
	CitiesStarving = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "cities_starving",
		Help:      "Owned cities currently starving.",
	})

	// UserFoodSum is the total food pooled across all users.
	UserFoodSum = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "user_food_sum",
		Help:      "Total food across all user pools.",
	})

	// UserGoldSum is the total gold across all users.
	UserGoldSum = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "user_gold_sum",
		Help:      "Total gold across all users.",
	})

	// FoodProductionRateSum is the per-hour food production summed across
	// owned cities.
	FoodProductionRateSum = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "food_production_rate_sum",
		Help:      "Per-hour food production across all owned cities.",
	})

	// FoodUpkeepRateSum is the per-hour food upkeep summed across owned cities.
	FoodUpkeepRateSum = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "food_upkeep_rate_sum",
		Help:      "Per-hour food upkeep across all owned cities.",
	})
)

// --- Game events ------------------------------------------------------------

var (
	// RegistrationsTotal counts registration attempts by outcome.
	RegistrationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "registrations_total",
		Help:      "User registration attempts by result.",
	}, []string{"result"}) // ok / invalid_argument / already_exists / internal

	// LoginsTotal counts login attempts by outcome.
	LoginsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "logins_total",
		Help:      "Login attempts by result.",
	}, []string{"result"}) // ok / invalid / not_found / internal

	// FoodDepositedTotal counts food units a city has handed to its user's pool.
	FoodDepositedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "food_deposited_total",
		Help:      "Total food deposited from city surpluses into user pools.",
	})

	// FoodWithdrawnTotal counts food units a city has pulled from its user's
	// pool to cover a local deficit.
	FoodWithdrawnTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "food_withdrawn_total",
		Help:      "Total food withdrawn from user pools to cover city deficits.",
	})

	// FoodPoolGrantsTotal counts food pool requests by how well the pool
	// covered them: full (granted >= requested), partial (granted > 0 but <
	// requested), or empty (granted == 0).
	FoodPoolGrantsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "food_pool_grants_total",
		Help:      "Food pool grant outcomes.",
	}, []string{"coverage"})

	// UpgradesStartedTotal counts building upgrade requests accepted, labelled
	// by building type and the target level.
	UpgradesStartedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "upgrades_started_total",
		Help:      "Building upgrade requests accepted (gold deducted, construction started).",
	}, []string{"building_type", "target_level"})

	// ConstructionCompletesTotal counts construction completions.
	ConstructionCompletesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "construction_completes_total",
		Help:      "Building construction completions, labelled by building type and final level.",
	}, []string{"building_type", "level"})

	// ConstructionDurationSeconds measures real-time elapsed during a
	// construction (from start to fire-time), to sanity-check that level-table
	// construction times are being honoured.
	ConstructionDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "construction_duration_seconds",
		Help:      "Construction wall-clock duration.",
		Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
	}, []string{"building_type"})
)

// --- Actor / runtime --------------------------------------------------------

var (
	// CityTickDurationSeconds measures how long tickFoodAndPopulation takes.
	CityTickDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "actor",
		Name:      "city_tick_duration_seconds",
		Help:      "Duration of one city tick (food loop + population).",
		Buckets:   prometheus.DefBuckets,
	})
)
