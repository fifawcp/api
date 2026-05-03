package app

import (
	"github.com/fifawcp/api/internal/middlewares"
	"github.com/go-chi/chi/v5"
)

func pickemRoutes(c *Container) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares.Auth(c.Authenticator, c.UserService, c.Logger))

	r.Get("/", c.PickemHandler.GetUserPickem)
	r.Put("/groups", c.PickemHandler.SaveGroupPicks)
	r.Put("/best-thirds", c.PickemHandler.SaveBestThirds)
	r.Put("/bracket", c.PickemHandler.SaveBracketPicks)

	return r
}
