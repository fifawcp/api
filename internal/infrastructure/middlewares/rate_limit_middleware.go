package middlewares

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/ratelimit"
	"github.com/fifawcp/api/internal/packages/httputils"
)

func RateLimitByIP(
	rl ratelimit.RateLimiter,
	scope string,
	logger logging.Logger,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If rate limiter is nil (RATE_LIMIT_ENABLED=false), pass through
			if rl == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Build key: middleware owns the <scope + ip> part,
			// the limiter appends the time-window bucket
			// Example: "otp_request:ip:203.0.113.42"
			ip := getClientIP(r)
			key := scope + ":ip:" + ip

			allowed, info, err := rl.Allow(r.Context(), key)
			if err != nil {
				// Fail open - log the error and let the request through
				// This could happen for example if the Redis connection fails
				logger.Error(
					"rate limiter error",
					"scope", scope,
					"ip", ip,
					"error", err,
				)

				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.Reset.Unix(), 10))

			if !allowed {
				// Seconds until the current window resets
				retryAfter := max(int64(time.Until(info.Reset).Seconds()), 0)
				w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))

				logger.Warn(
					"rate limit exceeded",
					"scope", scope,
					"ip", ip,
					"reset", info.Reset,
				)

				httputils.RespondWithError(w, http.StatusTooManyRequests, errors.New("rate limit exceeded"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
