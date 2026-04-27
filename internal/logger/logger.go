package logger

import (
	"log/slog"
	"os"
)

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
