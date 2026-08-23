// Package rpc implements the Connect RPC services, translating requests into
// actor messages and domain entities into their proto representations.
package rpc

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"cityio/internal/auth"
	"cityio/internal/contracts"
	"cityio/internal/domain"
	"cityio/internal/gen/cityio/service/v1/servicev1connect"
	"cityio/internal/messages"
	"cityio/internal/metrics"
)

// Server wires the Connect services to the actor cluster and persistence store.
type Server struct {
	cluster   contracts.ClusterProvider
	store     contracts.Store
	world     contracts.WorldProvider
	jwtSecret string

	// shutdownCtx is cancelled when the process is shutting down. Long-lived
	// handlers (StreamState) select on it and return Unauthenticated so clients
	// take their "session ended, log in again" path instead of seeing a
	// half-closed connection.
	shutdownCtx context.Context
}

// NewServer constructs an RPC server backed by the given cluster and store.
// shutdownCtx is cancelled by main on SIGINT/SIGTERM; streaming handlers
// observe it and close their streams.
func NewServer(shutdownCtx context.Context, cluster contracts.ClusterProvider, store contracts.Store, world contracts.WorldProvider, jwtSecret string) *Server {
	return &Server{cluster: cluster, store: store, world: world, jwtSecret: jwtSecret, shutdownCtx: shutdownCtx}
}

func (s *Server) ownedCities(ctx context.Context) ([]domain.City, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, errors.New("missing claims")
	}
	return s.store.GetCitiesByOwner(ctx, claims.UserID)
}

func (s *Server) liveArmies(ctx context.Context) ([]domain.Army, error) {
	armies, err := s.store.GetAllArmies(ctx)
	if err != nil {
		return nil, err
	}
	for i := range armies {
		res, err := s.cluster.Request("army", armies[i].ArmyID, messages.GetArmyMessage{})
		if err != nil {
			continue
		}
		if resp, ok := res.(*messages.GetArmyResponseMessage); ok {
			armies[i] = resp.Army
		}
	}
	return armies, nil
}

func (s *Server) liveCities(ctx context.Context) ([]domain.City, error) {
	cities, err := s.store.GetAllCities(ctx)
	if err != nil {
		return nil, err
	}
	for i := range cities {
		res, err := s.cluster.Request("city", cities[i].CityID, messages.GetCityMessage{})
		if err != nil {
			continue
		}
		if resp, ok := res.(*messages.GetCityResponseMessage); ok {
			cities[i] = resp.City
		}
	}
	return cities, nil
}

func (s *Server) liveBuildings(ctx context.Context) ([]domain.Building, error) {
	buildings, err := s.store.GetAllBuildings(ctx)
	if err != nil {
		return nil, err
	}
	for i := range buildings {
		res, err := s.cluster.Request("building", buildings[i].BuildingID, messages.GetBuildingMessage{})
		if err != nil {
			continue
		}
		if resp, ok := res.(*messages.GetBuildingResponseMessage); ok {
			buildings[i] = resp.Building
		}
	}
	return buildings, nil
}

func (s *Server) ownedVision(ctx context.Context) (domain.Vision, error) {
	armies, err := s.liveArmies(ctx)
	if err != nil {
		return domain.Vision{}, err
	}
	return s.ownedVisionWithArmies(ctx, armies)
}

func (s *Server) ownedVisionWithArmies(ctx context.Context, armies []domain.Army) (domain.Vision, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return domain.Vision{}, errors.New("missing claims")
	}
	cities, err := s.store.GetCitiesByOwner(ctx, claims.UserID)
	if err != nil {
		return domain.Vision{}, err
	}
	ownedArmies := make([]domain.Army, 0, len(armies))
	for _, army := range armies {
		if army.Owner == claims.UserID {
			ownedArmies = append(ownedArmies, army)
		}
	}
	return domain.Vision{Cities: cities, Armies: ownedArmies}, nil
}

func (s *Server) ownsCity(ctx context.Context, cityID string) (bool, error) {
	owned, err := s.ownedCities(ctx)
	if err != nil {
		return false, err
	}
	for _, c := range owned {
		if c.CityID == cityID {
			return true, nil
		}
	}
	return false, nil
}

// Handler returns the HTTP handler serving every Connect service with the
// metrics + auth interceptors applied (metrics is outermost so it captures
// auth failures and timing for them). /metrics and /healthz share the same
// listener as the Connect services — the nimbus deployment scrapes them off
// the API port rather than a dedicated metrics port.
func (s *Server) Handler() http.Handler {
	opts := connect.WithInterceptors(metrics.Interceptor(), auth.Interceptor(s.jwtSecret))

	mux := http.NewServeMux()
	mux.Handle(servicev1connect.NewUserServiceHandler(&userHandler{s}, opts))
	mux.Handle(servicev1connect.NewCityServiceHandler(&cityHandler{s}, opts))
	mux.Handle(servicev1connect.NewBuildingServiceHandler(&buildingHandler{s}, opts))
	mux.Handle(servicev1connect.NewArmyServiceHandler(&armyHandler{s}, opts))
	mux.Handle(servicev1connect.NewMapServiceHandler(&mapHandler{s}, opts))
	mux.Handle(servicev1connect.NewConfigServiceHandler(&configHandler{s}, opts))
	mux.Handle(servicev1connect.NewMailboxServiceHandler(&mailboxHandler{s}, opts))
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}
