// Package main provides the entry point for the micro-paas API server.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/masmuss/micro-paas/internal/app"
	"github.com/uptrace/bun/extra/bundebug"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, logger, db, err := app.Bootstrap(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv(""),
	))

	application, err := app.New(cfg, logger, db)
	if err != nil {
		return fmt.Errorf("create application: %w", err)
	}

	go func() {
		if startErr := application.Start(ctx); startErr != nil && !errors.Is(startErr, http.ErrServerClosed) {
			logger.ErrorContext(ctx, "Failed to start server", "error", startErr)
			stop()
		}
	}()

	<-ctx.Done()
	logger.InfoContext(ctx, "Shutting down server...")
	application.HealthChecker.Stop()
	return nil
}
