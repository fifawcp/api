package app

import (
	"github.com/fifawcp/api/internal/middlewares"
	"github.com/go-chi/chi/v5"
)

func dashboardRoutes(c *Container) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares.OptionalAuth(c.Authenticator, c.UserService, c.Logger))

	r.Get("/", c.DashboardHandler.GetDashboard)

	return r
}
