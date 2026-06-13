package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"cityio/internal/cluster"
	"cityio/internal/config"
	"cityio/internal/database"
	"cityio/internal/logger"
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
	cl := cluster.NewRuntime(ctx, db, cfg.Environment)

	setup.Run(ctx, &setup.Deps{
		DB:      db,
		Cluster: cl,
	})

	server := rpc.NewServer(cl, cfg.JWTSecret)
	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:4173", "http://localhost:3000", "https://cityio.prayujt.com"},
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

	slog.InfoContext(ctx, "serving connect rpc", "port", cfg.APIPort)
	if err := httpServer.ListenAndServe(); err != nil {
		slog.ErrorContext(ctx, "rpc server stopped", "error", err)
		os.Exit(1)
	}
}
