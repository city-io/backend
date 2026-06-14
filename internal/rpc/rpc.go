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
	"cityio/internal/ports"
)

// Server wires the Connect services to the actor cluster and persistence store.
type Server struct {
	cluster   ports.ClusterProvider
	store     ports.Store
	jwtSecret string
}

// NewServer constructs an RPC server backed by the given cluster and store.
func NewServer(cluster ports.ClusterProvider, store ports.Store, jwtSecret string) *Server {
	return &Server{cluster: cluster, store: store, jwtSecret: jwtSecret}
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

// Handler returns the HTTP handler serving every Connect service with the auth
// interceptor applied.
func (s *Server) Handler() http.Handler {
	opts := connect.WithInterceptors(auth.Interceptor(s.jwtSecret))

	mux := http.NewServeMux()
	mux.Handle(servicev1connect.NewUserServiceHandler(&userHandler{s}, opts))
	mux.Handle(servicev1connect.NewCityServiceHandler(&cityHandler{s}, opts))
	mux.Handle(servicev1connect.NewBuildingServiceHandler(&buildingHandler{s}, opts))
	mux.Handle(servicev1connect.NewMapServiceHandler(&mapHandler{s}, opts))
	mux.Handle(servicev1connect.NewConfigServiceHandler(&configHandler{s}, opts))
	return mux
}
