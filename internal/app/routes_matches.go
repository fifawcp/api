package app

import "github.com/go-chi/chi/v5"

func matchRoutes(c *Container) chi.Router {
	r := chi.NewRouter()
	r.Get("/", c.MatchHandler.GetMatches)
	return r
}
