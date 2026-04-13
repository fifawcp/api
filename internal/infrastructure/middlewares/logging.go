package middlewares

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/logging"
)

func LogRequest(logger logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t := time.Now()

			defer func() {
				logger.Info("Request processed",
					"method", r.Method,
					"path", r.URL.Path,
					"remote_addr", r.RemoteAddr,
					"user_agent", r.UserAgent(),
					"status", ww.Status(),
					"duration", time.Since(t).String(),
					"request_id", middleware.GetReqID(r.Context()),
				)
			}()

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
