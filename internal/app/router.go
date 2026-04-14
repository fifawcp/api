package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ncondes/fifawcp/internal/handlers"
	"github.com/ncondes/fifawcp/internal/infrastructure/middlewares"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (app *AppContainer) NewRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)                         // Add a request ID to the context
	r.Use(middleware.RealIP)                            // Get the real IP address of the client
	r.Use(middleware.Recoverer)                         // Recover from panics without crashing the server
	r.Use(middlewares.LogRequestMiddleware(app.Logger)) // Log requests

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(app.Config.Server.ContextTimeout))

	// Security headers
	r.Use(middlewares.SecurityHeadersMiddleware(app.Config))

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

	r.Route("/debug", func(r chi.Router) {
		if !app.Config.IsProd() {
			debugHandler := handlers.NewDebugHandler(app.Config)
			r.Get("/auth/otp/request/{identifier}", debugHandler.RequestOtp)
		}
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		r.Route("/auth", func(r chi.Router) {
			r.With(middlewares.RateLimitByIPMiddleware(
				app.RateLimiters.StrictIP,
				"auth:otp:request",
				app.Logger,
			)).Post("/otp/request", app.AuthHandler.RequestOtp)

			r.With(
				middlewares.RateLimitByIPMiddleware(
					app.RateLimiters.ModerateIP,
					"auth:token",
					app.Logger,
				),
				middlewares.RequestInfoMiddleware(),
			).Post("/token", app.AuthHandler.Authenticate)

			r.With(middlewares.RateLimitByIPMiddleware(
				app.RateLimiters.RelaxedIP,
				"auth:token:refresh",
				app.Logger,
			)).Post("/token/refresh", app.AuthHandler.RefreshToken)

			r.Post("/logout", app.AuthHandler.Logout)
			r.Post("/logout/all", app.AuthHandler.LogoutAll)
			r.Get("/sessions", app.AuthHandler.GetSessions)
			r.With(middlewares.AuthMiddleware(
				app.Authenticator,
				app.UserService,
				app.Logger,
			)).Delete("/sessions/{id}", app.AuthHandler.DeleteSession)
		})

		r.Route("/users", func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware(app.Authenticator, app.UserService, app.Logger))
			r.Get("/profile", app.UserHandler.GetProfile)
		})
	})

	return r
}
