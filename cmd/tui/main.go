// Package main provides the entry point for the micro-paas Terminal UI.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/masmuss/micro-paas/internal/app"
	"github.com/masmuss/micro-paas/internal/tui"
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

	// Redirect slog and other system logs to a file to avoid TUI corruption
	logFile, err := os.OpenFile("tui.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer logFile.Close()

	cfg, logger, db, err := app.Bootstrap(ctx, logFile)
	if err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	application, err := app.New(cfg, logger, db)
	if err != nil {
		return fmt.Errorf("create application: %w", err)
	}

	m := tui.NewModel(application.Repo, application.Pinger)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, runErr := p.Run(); runErr != nil {
		return fmt.Errorf("run TUI: %w", runErr)
	}

	return nil
}
