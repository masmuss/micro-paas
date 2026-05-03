// Package app contains the application entry and server setup.
package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/masmuss/micro-paas/internal/config"
	"github.com/masmuss/micro-paas/internal/database"
	"github.com/masmuss/micro-paas/internal/delivery/handler"
	"github.com/masmuss/micro-paas/internal/logger"
	"github.com/masmuss/micro-paas/internal/model"
	"github.com/masmuss/micro-paas/internal/repository"
	"github.com/masmuss/micro-paas/internal/service"
	"github.com/uptrace/bun"
)

// App is the main application container.
type App struct {
	Config        *config.Config
	Logger        *slog.Logger
	DB            *bun.DB
	Repo          repository.InstanceRepository
	Router        *chi.Mux
	Pinger        service.DockerService
	HealthChecker *service.HealthChecker
	UIHandler     *handler.UIHandler
}

// Bootstrap handles the initial setup (config, db, migrations) shared by all entry points.
func Bootstrap(ctx context.Context, logWriter io.Writer) (*config.Config, *slog.Logger, *bun.DB, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	log := logger.New(os.Getenv("APP_ENV") == "development", logWriter)

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load config: %w", err)
	}

	db, err := database.NewBunDB(cfg.DBDriver, cfg.DBDsn)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("connect db: %w", err)
	}

	_, err = db.NewCreateTable().Model((*model.Instance)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("migrate: %w", err)
	}

	return cfg, log, db, nil
}

// New creates and wires a new App instance. Returns error if dependency setup fails.
func New(cfg *config.Config, log *slog.Logger, db *bun.DB) (*App, error) {
	instanceRepo := repository.NewInstanceRepository(db, log)
	dockerSvc, err := service.NewDockerService(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("create docker service: %w", err)
	}

	instanceHandler := handler.NewInstanceHandler(dockerSvc, instanceRepo, log)
	uiHandler := handler.NewUIHandler(instanceRepo, dockerSvc, cfg)
	r := chi.NewRouter()

	healthChecker := service.NewHealthChecker(dockerSvc, instanceRepo, log, 30*time.Second)

	app := &App{
		Config:        cfg,
		Logger:        log,
		DB:            db,
		Repo:          instanceRepo,
		Router:        r,
		Pinger:        dockerSvc,
		HealthChecker: healthChecker,
		UIHandler:     uiHandler,
	}

	app.setupRoutes(instanceHandler, uiHandler)

	return app, nil
}

// Start runs the HTTP server with graceful shutdown support.
func (a *App) Start(ctx context.Context) error {
	const (
		rwTimeoutSec    = 10
		idleTimeoutSec  = 60
		shutdownTimeout = 5 * time.Second
	)

	if err := a.Pinger.Ping(ctx); err != nil {
		return fmt.Errorf("ping docker daemon during startup: %w", err)
	}

	a.HealthChecker.Start(ctx)

	srv := &http.Server{
		Addr:         ":" + a.Config.ServerPort,
		Handler:      a.Router,
		ReadTimeout:  time.Duration(rwTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(rwTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(idleTimeoutSec) * time.Second,
	}

	// Channel to listen for errors from ListenAndServe
	serverErrors := make(chan error, 1)

	go func() {
		a.Logger.InfoContext(ctx, "Server is starting", "port", a.Config.ServerPort)
		serverErrors <- srv.ListenAndServe()
	}()

	// Blocking select to wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case <-ctx.Done():
		a.Logger.InfoContext(ctx, "Graceful shutdown initiated...")

		a.HealthChecker.Stop()

		// Create a separate context for shutdown to ensure it has time to finish
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			// If shutdown fails, force close
			_ = srv.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}

		if err := a.DB.Close(); err != nil {
			return fmt.Errorf("could not close database: %w", err)
		}

		a.Logger.InfoContext(ctx, "Server stopped gracefully")
	}

	return nil
}
