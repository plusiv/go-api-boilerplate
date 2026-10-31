// Package logger provides request-scoped structured loggers.
package logger

import (
	"context"
	"log/slog"
)

type contextKey struct{}

// WithContext attaches a logger to ctx while preserving its cancellation and values.
func WithContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, log)
}

// FromContext returns the attached logger, or the default logger if none exists.
func FromContext(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(contextKey{}).(*slog.Logger); ok && log != nil {
		return log
	}
	return slog.Default()
}
