package app

import (
	"github.com/fifawcp/api/internal/infrastructure/middlewares"
	"github.com/go-chi/chi/v5"
)

func usersRoutes(c *Container) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares.Auth(c.Authenticator, c.UserService, c.Logger))

	r.Get("/profile", c.UserHandler.GetProfile)

	return r
}
