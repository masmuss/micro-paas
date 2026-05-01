// Package main provides the entry point for the micro-paas API server.
package main

import (
	"context"
	"fmt"
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
	// 1. Create a parent context that listens for termination signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Initialize core dependencies
	cfg, logger, db, err := app.Bootstrap(ctx, os.Stdout)
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	// Optional debug hooks
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv(""),
	))

	// Initialize Application
	application, err := app.New(cfg, logger, db)
	if err != nil {
		return fmt.Errorf("create application: %w", err)
	}

	// Start the server
	return application.Start(ctx)
}
