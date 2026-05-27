package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/fifawcp/api/internal/httpctx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/go-chi/chi/v5/middleware"
)

func LogRequest(logger logging.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Intercepts WriteHeader and Write calls to capture the status code and response body
			wrapResponseWriter := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			// Shared pointer: handlers and httpx mutate this when sending errors
			responseErr := &httpctx.ResponseError{}

			// Enriches the logger with request-specific fields
			enriched := logger.With(
				logging.RequestID, middleware.GetReqID(r.Context()),
				logging.Method, r.Method,
				logging.Path, r.URL.Path,
			)

			ctx := context.WithValue(r.Context(), httpctx.ResponseErrorContextKey, responseErr)
			r = r.WithContext(ctx)

			// Run after ServeHTTP returns
			defer func() {
				status := wrapResponseWriter.Status()

				outcome := "success"
				switch {
				case status >= 500:
					outcome = "server_error"
				case status >= 400:
					outcome = "user_error"
				}

				fields := []any{
					logging.IP, getClientIP(r),
					logging.Status, status,
					logging.DurationMS, time.Since(start).Milliseconds(),
					logging.Outcome, outcome,
				}

				// Append user ID if the endpoint is authenticated
				if user := httpctx.GetAuthenticatedUser(r.Context()); user != nil {
					fields = append(fields, logging.UserID, user.ID)
				}

				// If there's an error, add the error code and message
				if responseErr.Code != "" {
					fields = append(fields,
						"error_code", responseErr.Code,
						"error_message", responseErr.Message,
					)
				}

				enriched.Info("http request completed", fields...)
			}()

			next.ServeHTTP(wrapResponseWriter, r)
		})
	}
}
