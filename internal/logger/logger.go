// Package logger provides logging utilities for the application.
package logger

import (
	"io"
	"log/slog"
	"os"
)

// New returns a new [slog.Logger] configured for development (text, debug) or production (JSON, info) mode.
// Set isDebug to true for verbose output during development, or false for structured logs in production.
// w specifies where the logs should be written (defaults to [os.Stdout] if nil).
func New(isDebug bool, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}

	var handler slog.Handler

	if isDebug {
		handler = slog.NewTextHandler(w, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	return slog.New(handler)
}
