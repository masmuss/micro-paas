// Package app contains the application entry and server setup.
package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/masmuss/micro-paas/internal/config"
	"github.com/masmuss/micro-paas/internal/handler"
	"github.com/masmuss/micro-paas/internal/repository"
	"github.com/uptrace/bun"
)

// App is the main application container.
type App struct {
	Config *config.Config
	Logger *slog.Logger
	DB     *bun.DB
	Router *chi.Mux
}

// New returns a new App.
func New(cfg *config.Config, log *slog.Logger, db *bun.DB) *App {
	instanceRepo := repository.NewInstanceRepository(db, log)
	instanceHandler := handler.NewInstanceHandler(instanceRepo, log)

	r := chi.NewRouter()

	app := &App{
		Config: cfg,
		Logger: log,
		DB:     db,
		Router: r,
	}

	app.setupRoutes(instanceHandler)

	return app
}

// Start runs the HTTP server.
func (a *App) Start() error {
       a.Logger.Info("Server is starting", "port", a.Config.ServerPort)
       rwTimeoutSec := 10
       idleTimeoutSec := 60

       srv := &http.Server{
	       Addr:         ":" + a.Config.ServerPort,
	       Handler:      a.Router,
	       ReadTimeout:  time.Duration(rwTimeoutSec) * time.Second,
	       WriteTimeout: time.Duration(rwTimeoutSec) * time.Second,
	       IdleTimeout:  time.Duration(idleTimeoutSec) * time.Second,
       }
       return srv.ListenAndServe()
}
