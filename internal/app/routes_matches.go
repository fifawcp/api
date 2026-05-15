package app

import (
	"github.com/fifawcp/api/internal/middlewares"
	"github.com/go-chi/chi/v5"
)

func matchRoutes(c *Container) chi.Router {
	r := chi.NewRouter()

	// GET /matches embeds the caller's user_pick when authenticated
	// OptionalAuth lets anonymous callers through with no pick attached
	r.With(middlewares.OptionalAuth(c.Authenticator, c.UserService, c.Logger)).
		Get("/", c.MatchHandler.GetMatches)

	r.With(middlewares.Auth(c.Authenticator, c.UserService, c.Logger)).
		With(middlewares.ParseMatchID).
		Put("/{id}/pick", c.MatchHandler.SaveMatchScorePick)

	return r
}
