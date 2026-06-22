package metrics

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// DefaultAddr is the listen address for the metrics endpoint. Internal only —
// no auth is applied; deploy behind a private network.
const DefaultAddr = ":9090"

// Serve starts an HTTP server that exposes /metrics on addr. It runs in a
// background goroutine and shuts down when shutdownCtx is cancelled. Errors
// other than ErrServerClosed are logged but not returned (the metrics endpoint
// dying shouldn't crash the game server).
func Serve(shutdownCtx context.Context, addr string) {
	if addr == "" {
		addr = DefaultAddr
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		slog.InfoContext(shutdownCtx, "starting metrics server", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(shutdownCtx, "metrics server stopped", "error", err)
		}
	}()

	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()
}
