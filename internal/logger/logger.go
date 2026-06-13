// Package logger configures the global slog logger with a context-aware
// handler. Attributes attached to a context with With are automatically
// emitted by every slog call that receives that context (for example
// slog.InfoContext).
package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

// Setup installs a context-aware tint handler as the global slog default at
// the provided level.
func Setup(level slog.Level) {
	handler := &contextHandler{
		Handler: tint.NewHandler(os.Stderr, &tint.Options{
			Level:      level,
			TimeFormat: time.Kitchen,
		}),
	}
	slog.SetDefault(slog.New(handler))
}
