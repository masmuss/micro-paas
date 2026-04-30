package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/masmuss/micro-paas/internal/delivery/handler"
)

func (a *App) setupRoutes(instanceHandler *handler.InstanceHandler) {
	a.Router.Use(middleware.RequestID)
	a.Router.Use(middleware.RealIP)
	a.Router.Use(middleware.Logger)
	a.Router.Use(middleware.Recoverer)

	a.Router.Route("/api", func(r chi.Router) {
		r.Route("/instances", func(r chi.Router) {
			r.Get("/", instanceHandler.List)
			r.Post("/", instanceHandler.Create)
			r.Delete("/{id}", instanceHandler.Delete)
		})
	})
}
