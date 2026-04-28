// Package logger provides logging utilities for the application.
package logger

import (
	"log/slog"
	"os"
)

// New returns a new [slog.Logger] configured for development (text, debug) or production (JSON, info) mode.
// Set isDebug to true for verbose output during development, or false for structured logs in production.
func New(isDebug bool) *slog.Logger {
	var handler slog.Handler

	if isDebug {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	return slog.New(handler)
}
