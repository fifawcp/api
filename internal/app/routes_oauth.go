package app

import (
	"github.com/fifawcp/api/internal/middlewares"
	"github.com/go-chi/chi/v5"
)

func oauthRoutes(c *Container) chi.Router {
	r := chi.NewRouter()

	r.With(
		middlewares.RateLimitByIP(c.RateLimiters.StrictIP, "oauth:google", c.Logger),
		middlewares.RequireOAuthReturnTo(c.Logger, c.Config.Auth.GoogleOAuth.ReturnToAllowlist),
	).Get("/google", c.OAuthHandler.GoogleOAuth)

	r.With(
		middlewares.RateLimitByIP(c.RateLimiters.ModerateIP, "oauth:google:callback", c.Logger),
		middlewares.ValidateOAuthCallback(c.Logger),
		middlewares.RequestInfo(),
	).Get("/google/callback", c.OAuthHandler.GoogleOAuthCallback)

	return r
}
