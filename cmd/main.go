// Package main is the entry point for the micro-paas application.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/masmuss/micro-paas/internal/app"
	"github.com/masmuss/micro-paas/internal/config"
	"github.com/masmuss/micro-paas/internal/database"
	"github.com/masmuss/micro-paas/internal/logger"
	"github.com/masmuss/micro-paas/internal/model"
	"github.com/uptrace/bun/extra/bundebug"
)

// main is the entry point for the application.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	log := logger.New(os.Getenv("APP_ENV") == "development")

	cfg, cfgErr := config.LoadConfig()
	if cfgErr != nil {
		log.ErrorContext(ctx, "Failed to load config", "error", cfgErr)
		stop()
		os.Exit(1)
		return
	}

	db, dbErr := database.NewBunDB(cfg.DBPath)
	if dbErr != nil {
		log.ErrorContext(ctx, "Failed to connect to database", "error", dbErr)
		stop()
		os.Exit(1)
		return
	}

	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv(""),
	))

	_, tableErr := db.NewCreateTable().Model((*model.Instance)(nil)).IfNotExists().Exec(ctx)
	if tableErr != nil {
		log.ErrorContext(ctx, "Failed to create instances table", "error", tableErr)
		stop()
		os.Exit(1)
		return
	}

	application, err := app.New(cfg, log, db)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create application", "error", err)
		stop()
		os.Exit(1)
		return
	}

	go func() {
		if startErr := application.Start(ctx); startErr != nil && !errors.Is(startErr, http.ErrServerClosed) {
			log.ErrorContext(ctx, "Failed to start server", "error", startErr)
			stop()
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	stop()

	log.InfoContext(ctx, "Shutting down micro-paas server")

	seconds := 5
	_, cancel := context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
	defer cancel()

	log.InfoContext(ctx, "Server stopped")
}
