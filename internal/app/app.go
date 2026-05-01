// Package app contains the application entry and server setup.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/masmuss/micro-paas/internal/config"
	"github.com/masmuss/micro-paas/internal/delivery/handler"
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
}

// New creates and wires a new App instance. Returns error if dependency setup fails.
func New(cfg *config.Config, log *slog.Logger, db *bun.DB) (*App, error) {
	instanceRepo := repository.NewInstanceRepository(db, log)
	dockerSvc, err := service.NewDockerService(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("create docker service: %w", err)
	}

	instanceHandler := handler.NewInstanceHandler(dockerSvc, instanceRepo, log)
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
	}

	app.setupRoutes(instanceHandler)

	return app, nil
}

// Start runs the HTTP server and pings Docker daemon before serving.
func (a *App) Start(ctx context.Context) error {
	const (
		rwTimeoutSec   = 10
		idleTimeoutSec = 60
	)
	if err := a.Pinger.Ping(ctx); err != nil {
		return fmt.Errorf("ping docker daemon during startup: %w", err)
	}

	a.HealthChecker.Start(ctx)

	a.Logger.InfoContext(ctx, "Server is starting", "port", a.Config.ServerPort)

	srv := &http.Server{
		Addr:         ":" + a.Config.ServerPort,
		Handler:      a.Router,
		ReadTimeout:  time.Duration(rwTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(rwTimeoutSec) * time.Second,
		IdleTimeout:  time.Duration(idleTimeoutSec) * time.Second,
	}
	return srv.ListenAndServe()
}
