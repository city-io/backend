package logger

import "context"

type contextKey struct{}

// WithContext attaches the provided logger to the context.
func WithContext(ctx context.Context, log Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, log)
}

// FromContext retrieves a logger from the context, returning nil when not present.
func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		return nil
	}
	if log, ok := ctx.Value(contextKey{}).(Logger); ok {
		return log
	}

	return nil
}
