package app

import (
	"github.com/fifawcp/api/internal/middlewares"
	"github.com/go-chi/chi/v5"
)

func awardRoutes(c *Container) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares.Auth(c.Authenticator, c.UserService, c.Logger))

	r.Get("/", c.AwardHandler.GetUserAwards)
	r.Get("/popular", c.AwardHandler.GetPopularPicks)
	r.Put("/", c.AwardHandler.SaveAwardPicks)

	return r
}
