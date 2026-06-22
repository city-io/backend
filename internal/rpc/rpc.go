// Package rpc implements the Connect RPC services, translating requests into
// actor messages and domain entities into their proto representations.
package rpc

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"cityio/internal/auth"
	"cityio/internal/domain"
	"cityio/internal/gen/cityio/service/v1/servicev1connect"
	"cityio/internal/metrics"
	"cityio/internal/ports"
)

// Server wires the Connect services to the actor cluster and persistence store.
type Server struct {
	cluster   ports.ClusterProvider
	store     ports.Store
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
func NewServer(shutdownCtx context.Context, cluster ports.ClusterProvider, store ports.Store, jwtSecret string) *Server {
	return &Server{cluster: cluster, store: store, jwtSecret: jwtSecret, shutdownCtx: shutdownCtx}
}

func (s *Server) ownedCities(ctx context.Context) ([]domain.City, error) {
	claims, ok := auth.ClaimsFromContext(ctx)
	if !ok {
		return nil, errors.New("missing claims")
	}
	return s.store.GetCitiesByOwner(ctx, claims.UserID)
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
// auth failures and timing for them).
func (s *Server) Handler() http.Handler {
	opts := connect.WithInterceptors(metrics.Interceptor(), auth.Interceptor(s.jwtSecret))

	mux := http.NewServeMux()
	mux.Handle(servicev1connect.NewUserServiceHandler(&userHandler{s}, opts))
	mux.Handle(servicev1connect.NewCityServiceHandler(&cityHandler{s}, opts))
	mux.Handle(servicev1connect.NewBuildingServiceHandler(&buildingHandler{s}, opts))
	mux.Handle(servicev1connect.NewMapServiceHandler(&mapHandler{s}, opts))
	mux.Handle(servicev1connect.NewConfigServiceHandler(&configHandler{s}, opts))
	return mux
}
