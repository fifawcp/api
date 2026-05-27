package app

import (
	"github.com/fifawcp/api/internal/handlers"
	"github.com/go-chi/chi/v5"
)

func debugRoutes(c *Container) chi.Router {
	r := chi.NewRouter()

	debugHandler := handlers.NewDebugHandler(c.Config)
	r.Get("/totp/{identifier}", debugHandler.RequestTotp)

	return r
}
