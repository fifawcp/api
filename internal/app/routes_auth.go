package app

import (
	"github.com/fifawcp/api/internal/middlewares"
	"github.com/go-chi/chi/v5"
)

func authRoutes(c *Container) chi.Router {
	r := chi.NewRouter()

	r.With(middlewares.RateLimitByIP(c.RateLimiters.StrictIP, "auth:otp:request", c.Logger)).
		Post("/otp/request", c.AuthHandler.RequestOtp)

	r.With(
		middlewares.RateLimitByIP(c.RateLimiters.ModerateIP, "auth:otp:verify", c.Logger),
		middlewares.RequestInfo(),
	).Post("/otp/verify", c.AuthHandler.VerifyOtp)

	r.With(
		middlewares.RateLimitByIP(c.RateLimiters.ModerateIP, "auth:token", c.Logger),
		middlewares.RequestInfo(),
	).Post("/token", c.AuthHandler.Authenticate)

	r.With(middlewares.RateLimitByIP(c.RateLimiters.RelaxedIP, "auth:token:refresh", c.Logger)).
		Post("/token/refresh", c.AuthHandler.RefreshToken)

	r.Post("/logout", c.AuthHandler.Logout)
	r.Post("/logout/all", c.AuthHandler.LogoutAll)
	r.Get("/sessions", c.AuthHandler.GetSessions)

	r.With(middlewares.Auth(c.Authenticator, c.UserService, c.Logger)).
		Delete("/sessions/{id}", c.AuthHandler.DeleteSession)

	return r
}
