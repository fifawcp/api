package app

import (
	"net/http"

	"github.com/fifawcp/api/internal/handlers"
	"github.com/fifawcp/api/internal/infrastructure/middlewares"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (app *AppContainer) NewRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)               // Add a request ID to the context
	r.Use(middleware.RealIP)                  // Get the real IP address of the client
	r.Use(middleware.Recoverer)               // Recover from panics without crashing the server
	r.Use(middlewares.LogRequest(app.Logger)) // Log requests

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(app.Config.Server.ContextTimeout))

	// Security headers
	r.Use(middlewares.SecurityHeaders(app.Config))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   app.Config.Server.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	// TODO: Metrics

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(app.Config.APIBaseURL+"/swagger/doc.json"),
	))

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		r.Route("/auth", func(r chi.Router) {
			r.With(middlewares.RateLimitByIP(
				app.RateLimiters.StrictIP,
				"auth:otp:request",
				app.Logger,
			)).Post("/otp/request", app.AuthHandler.RequestOtp)
			r.With(
				middlewares.RateLimitByIP(
					app.RateLimiters.ModerateIP,
					"auth:otp:verify",
					app.Logger,
				),
				middlewares.RequestInfo(),
			).Post("/otp/verify", app.AuthHandler.VerifyOtp)
			r.With(
				middlewares.RateLimitByIP(
					app.RateLimiters.ModerateIP,
					"auth:token",
					app.Logger,
				),
				middlewares.RequestInfo(),
			).Post("/token", app.AuthHandler.Authenticate)

			r.With(middlewares.RateLimitByIP(
				app.RateLimiters.RelaxedIP,
				"auth:token:refresh",
				app.Logger,
			)).Post("/token/refresh", app.AuthHandler.RefreshToken)

			r.Post("/logout", app.AuthHandler.Logout)
			r.Post("/logout/all", app.AuthHandler.LogoutAll)
			r.Get("/sessions", app.AuthHandler.GetSessions)

			r.With(middlewares.Auth(
				app.Authenticator,
				app.UserService,
				app.Logger,
			)).Delete("/sessions/{id}", app.AuthHandler.DeleteSession)
		})

		r.Route("/oauth", func(r chi.Router) {
			r.With(
				middlewares.RateLimitByIP(
					app.RateLimiters.StrictIP,
					"oauth:google",
					app.Logger,
				),
				middlewares.RequireOAuthReturnTo(
					app.Logger,
					app.Config.Auth.GoogleOAuth.ReturnToAllowlist,
				),
			).Get("/google", app.OAuthHandler.GoogleOAuth)

			r.With(
				middlewares.RateLimitByIP(
					app.RateLimiters.ModerateIP,
					"oauth:google:callback",
					app.Logger,
				),
				middlewares.ValidateOAuthCallback(app.Logger),
				middlewares.RequestInfo(),
			).Get("/google/callback", app.OAuthHandler.GoogleOAuthCallback)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(middlewares.Auth(
				app.Authenticator,
				app.UserService,
				app.Logger,
			))
			r.Use(middlewares.RequireAdminRole)

			r.Route("/matches", func(r chi.Router) {
				r.Post("/results", app.AdminHandler.BulkUpdateMatchResults)

				r.Route("/{id}", func(r chi.Router) {
					r.Use(middlewares.RequireValidMatchID(app.Logger))
					r.Post("/result", app.AdminHandler.UpdateMatchResult)
					r.Delete("/result", app.AdminHandler.ResetMatchResult)
				})
			})

			r.Route("/standings", func(r chi.Router) {
				r.Post("/recalculate", app.AdminHandler.RecalculateStandings)

				r.Route("/third-place", func(r chi.Router) {
					r.Post("/resolve", app.AdminHandler.ResolveThirdPlaceConflict)
				})
			})
		})

		r.Route("/standings", func(r chi.Router) {
			r.Get("/", app.GroupHandler.GetGroupStandings)
		})

		r.Route("/matches", func(r chi.Router) {
			r.Get("/", app.MatchHandler.GetMatches)
		})

		r.Route("/boards", func(r chi.Router) {
			r.Use(middlewares.Auth(
				app.Authenticator,
				app.UserService,
				app.Logger,
			))

			r.Get("/", app.BoardHandler.GetUserBoards)
			r.Post("/", app.BoardHandler.CreateBoard)
			r.Post("/join", app.BoardHandler.JoinBoard)

			r.Route("/{boardId}", func(r chi.Router) {
				r.Use(middlewares.RequireBoardMembership(
					app.BoardMemberService,
				))

				r.Get("/", app.BoardHandler.GetBoardByID)
				r.Patch("/", app.BoardHandler.UpdateBoard)
				r.Delete("/", app.BoardHandler.DeleteBoard)
				r.Get("/ranking", app.BoardHandler.GetBoardRanking)
				r.Post("/regenerate-join-code", app.BoardHandler.RegenerateJoinCode)

				r.Route("/members", func(r chi.Router) {
					r.Get("/", app.BoardHandler.GetBoardMembers)
					r.With(
						middlewares.RequireValidUserID,
					).Patch("/{userId}/role", app.BoardHandler.UpdateBoardMemberRole)
					r.With(
						middlewares.RequireValidUserID,
					).Delete("/{userId}", app.BoardHandler.RemoveBoardMember)
				})
			})
		})

		r.Route("/users", func(r chi.Router) {
			r.Use(middlewares.Auth(app.Authenticator, app.UserService, app.Logger))
			r.Get("/profile", app.UserHandler.GetProfile)
		})

		r.Route("/debug", func(r chi.Router) {
			if !app.Config.IsProd() {
				debugHandler := handlers.NewDebugHandler(app.Config)
				r.Get("/totp/{identifier}", debugHandler.RequestTotp)
			}
		})
	})

	return r
}
