package client

import "context"

// Logger is a pluggable logging interface for the client package.
// Implementations can bridge to any logging framework (tflog, slog, etc).
type Logger interface {
	Debug(ctx context.Context, msg string, fields map[string]any)
}

// NopLogger is a no-op logger that discards all messages.
type NopLogger struct{}

func (NopLogger) Debug(context.Context, string, map[string]any) {}
