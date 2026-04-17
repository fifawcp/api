package middlewares

import (
	"net/http"

	"github.com/fifawcp/api/internal/infrastructure/config"
)

func SecurityHeaders(cfg *config.Config) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Browser MIME-type sniffing protection
			w.Header().Set("X-Content-Type-Options", "nosniff")
			// Clickjacking protection
			w.Header().Set("X-Frame-Options", "DENY")
			// XSS protection
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			// Referrer policy - Leaking URL info to third parties
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			if cfg.IsProd() {
				// HSTS - Force HTTPS
				w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}
