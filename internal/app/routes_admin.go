package app

import (
	"github.com/fifawcp/api/internal/middlewares"
	"github.com/go-chi/chi/v5"
)

func adminRoutes(c *Container) chi.Router {
	r := chi.NewRouter()
	r.Use(middlewares.Auth(c.Authenticator, c.UserService, c.Logger))
	r.Use(middlewares.RequireAdminRole)

	r.Route("/matches", func(r chi.Router) {
		r.Post("/results", c.AdminHandler.BulkUpdateMatchResults)

		r.Route("/{id}", func(r chi.Router) {
			r.Use(middlewares.ParseMatchID)
			r.Post("/result", c.AdminHandler.UpdateMatchResult)
			r.Delete("/result", c.AdminHandler.ResetMatchResult)
		})
	})

	r.Route("/standings", func(r chi.Router) {
		r.Post("/recalculate", c.AdminHandler.RecalculateStandings)

		r.Route("/third-place", func(r chi.Router) {
			r.Post("/resolve", c.AdminHandler.ResolveThirdPlaceConflict)
		})
	})

	r.Route("/pickems", func(r chi.Router) {
		r.Post("/rescore/best-thirds", c.AdminHandler.RescoreBestThirds)

		r.Route("/rescore/match/{id}", func(r chi.Router) {
			r.Use(middlewares.ParseMatchID)
			r.Post("/", c.AdminHandler.RescoreMatch)
		})
	})

	r.Route("/awards", func(r chi.Router) {
		r.Post("/winners", c.AdminHandler.RecordAwardWinners)
	})

	return r
}
