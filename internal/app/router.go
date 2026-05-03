package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/masmuss/micro-paas/internal/delivery/handler"
	paasMiddleware "github.com/masmuss/micro-paas/internal/delivery/middleware"
)

func (a *App) setupRoutes(instanceHandler *handler.InstanceHandler, uiHandler *handler.UIHandler) {
	proxy := paasMiddleware.NewInstanceProxy(a.Repo, a.Logger, a.Config)

	a.Router.Use(middleware.RequestID)
	a.Router.Use(middleware.RealIP)
	a.Router.Use(middleware.Logger)
	a.Router.Use(middleware.Recoverer)
	a.Router.Use(proxy.Handler)

	// UI Routes
	a.Router.Get("/", uiHandler.Dashboard)
	a.Router.Get("/ui/instances-table", uiHandler.InstancesTable)
	a.Router.Post("/ui/instances/{id}/start", uiHandler.StartInstance)
	a.Router.Post("/ui/instances/{id}/stop", uiHandler.StopInstance)
	a.Router.Delete("/ui/instances/{id}", uiHandler.DeleteInstance)
	a.Router.Post("/ui/instances", uiHandler.CreateInstance)
	a.Router.Get("/ui/instances/{id}/logs", uiHandler.LogsModal)

	a.Router.Route("/api", func(r chi.Router) {
		r.Route("/instances", func(r chi.Router) {
			r.Get("/", instanceHandler.List)
			r.Post("/", instanceHandler.Create)
			r.Get("/{id}", instanceHandler.GetByID)
			r.Patch("/{id}", instanceHandler.Update)
			r.Post("/{id}/start", instanceHandler.Start)
			r.Post("/{id}/stop", instanceHandler.Stop)
			r.Post("/{id}/restart", instanceHandler.Restart)
			r.Get("/{id}/logs", instanceHandler.Logs)
			r.Delete("/{id}", instanceHandler.Delete)
		})
	})
}
