package app

import "github.com/go-chi/chi/v5"

func standingsRoutes(c *Container) chi.Router {
	r := chi.NewRouter()
	r.Get("/", c.GroupHandler.GetGroupStandings)
	return r
}
