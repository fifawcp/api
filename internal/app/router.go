package app

import (
	"net/http"

	"github.com/fifawcp/api/internal/middlewares"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (c *Container) NewRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middlewares.TrustedProxyRealIP(c.Config.Server.TrustedProxyCIDRs))
	r.Use(middleware.Recoverer)
	r.Use(middlewares.LogRequest(c.Logger))
	r.Use(middleware.Timeout(c.Config.Server.ContextTimeout))
	r.Use(middlewares.SecurityHeaders(c.Config))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   c.Config.Server.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(c.Config.APIBaseURL+"/swagger/doc.json"),
	))

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		r.Mount("/auth", authRoutes(c))
		r.Mount("/oauth", oauthRoutes(c))
		r.Mount("/admin", adminRoutes(c))

		r.Mount("/standings", standingsRoutes(c))
		r.Mount("/matches", matchRoutes(c))

		r.Mount("/boards", boardsRoutes(c))
		r.Mount("/users", usersRoutes(c))
		r.Mount("/pickems", pickemRoutes(c))

		if !c.Config.IsProd() {
			r.Mount("/debug", debugRoutes(c))
		}
	})

	return r
}
