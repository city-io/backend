package logger

import (
	"context"
	"log/slog"
	"time"
)

type ctxKey struct{}

// With returns a copy of ctx carrying the supplied key/value attributes in
// addition to any already present. The attributes are emitted by every
// context-aware slog call (slog.InfoContext, slog.ErrorContext, ...) made with
// the returned context.
func With(ctx context.Context, args ...any) context.Context {
	attrs := argsToAttrs(args)
	if len(attrs) == 0 {
		return ctx
	}

	existing := fromContext(ctx)
	merged := make([]slog.Attr, 0, len(existing)+len(attrs))
	merged = append(merged, existing...)
	merged = append(merged, attrs...)
	return context.WithValue(ctx, ctxKey{}, merged)
}

func fromContext(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	if attrs, ok := ctx.Value(ctxKey{}).([]slog.Attr); ok {
		return attrs
	}
	return nil
}

// argsToAttrs converts alternating key/value pairs into slog attributes,
// reusing slog's own argument normalization.
func argsToAttrs(args []any) []slog.Attr {
	if len(args) == 0 {
		return nil
	}
	r := slog.NewRecord(time.Time{}, 0, "", 0)
	r.Add(args...)
	attrs := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	return attrs
}

// contextHandler decorates a slog.Handler, injecting attributes stored on the
// context into every record before delegating to the wrapped handler.
type contextHandler struct {
	slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs := fromContext(ctx); len(attrs) > 0 {
		r.AddAttrs(attrs...)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithGroup(name)}
}
