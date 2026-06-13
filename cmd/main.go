package main

import (
	"context"
	"log/slog"
	"os"

	"cityio/internal/api"
	"cityio/internal/cluster"
	"cityio/internal/config"
	"cityio/internal/database"
	"cityio/internal/logger"
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

	api.Start(cfg.APIPort)
}
