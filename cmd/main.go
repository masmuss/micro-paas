package main

import (
	"os"

	"github.com/masmuss/micro-paas/internal/config"
	"github.com/masmuss/micro-paas/internal/database"
	"github.com/masmuss/micro-paas/internal/logger"
	"github.com/uptrace/bun/extra/bundebug"
)

func main() {
	log := logger.New(os.Getenv("APP_ENV") == "development")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := database.NewBunDB(cfg.DBPath)
	if err != nil {
		log.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv(""),
	))

	log.Info("Starting micro-paas server", "port", cfg.ServerPort)
}
