package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"cityio/internal/cluster"
	"cityio/internal/config"
	"cityio/internal/database"
	"cityio/internal/logger"
	"cityio/internal/metrics"
	"cityio/internal/persistence"
	"cityio/internal/rpc"
	"cityio/internal/setup"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	level := slog.LevelDebug
	if cfg.IsProduction() {
		level = slog.LevelInfo
	}
	logger.Setup(level)

	ctx := logger.With(context.Background(), "environment", cfg.Environment)
	slog.InfoContext(ctx, "starting cityio backend")

	db := database.NewDB(ctx, cfg.DatabaseDSN())
	store := persistence.New(db)
	store.Start(ctx)
	cl := cluster.NewRuntime(ctx, store, cfg.Environment)

	setup.Run(ctx, &setup.Deps{
		DB:      db,
		Cluster: cl,
	})

	// shutdownCtx is cancelled when we receive SIGINT/SIGTERM. The RPC server
	// hands it to long-lived handlers (StreamState) so they can close cleanly
	// instead of dying mid-connection.
	shutdownCtx, cancelShutdown := context.WithCancel(ctx)
	defer cancelShutdown()

	// Internal-only metrics endpoint + periodic state snapshot. No auth —
	// scrape from the private network.
	metrics.Serve(shutdownCtx, metrics.DefaultAddr)
	metrics.StartSnapshot(shutdownCtx, store)

	server := rpc.NewServer(shutdownCtx, cl, store, cfg.JWTSecret)
	handler := cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool {
			if origin == "http://localhost:5173" || origin == "http://localhost:4173" {
				return true
			}
			return strings.HasSuffix(origin, ".prayujt.com")
		},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
		// Connect-specific headers must be exposed for streaming/error metadata.
		ExposedHeaders: []string{"Connect-Protocol-Version", "Connect-Timeout-Ms", "Grpc-Status", "Grpc-Message", "Grpc-Status-Details-Bin"},
	}).Handler(server.Handler())

	httpServer := &http.Server{
		Addr:        fmt.Sprintf("0.0.0.0:%s", cfg.APIPort),
		Handler:     h2c.NewHandler(handler, &http2.Server{}),
		ReadTimeout: 15 * time.Second,
	}

	// Catch SIGINT/SIGTERM, signal active streams to close, then drain HTTP
	// and the persistence flush queue before exiting.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.InfoContext(ctx, "shutdown signal received", "signal", sig.String())
		cancelShutdown()

		// Give in-flight StreamState handlers a moment to observe the
		// cancellation and return Unauthenticated to their clients.
		shutdownTimeout, cancelTimeout := context.WithTimeout(ctx, 10*time.Second)
		defer cancelTimeout()
		if err := httpServer.Shutdown(shutdownTimeout); err != nil {
			slog.ErrorContext(ctx, "http server shutdown error", "error", err)
		}
		store.Stop(ctx)
	}()

	slog.InfoContext(ctx, "serving connect rpc", "port", cfg.APIPort)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.ErrorContext(ctx, "rpc server stopped", "error", err)
		os.Exit(1)
	}
	slog.InfoContext(ctx, "rpc server stopped cleanly")
}
