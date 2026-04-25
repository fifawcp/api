package middlewares

import (
	"net/http"
	"strconv"
	"time"

	"github.com/fifawcp/api/internal/httpx"
	"github.com/fifawcp/api/internal/infrastructure/logging"
	"github.com/fifawcp/api/internal/infrastructure/ratelimit"
)

func RateLimitByIP(
	rl ratelimit.RateLimiter,
	scope string,
	logger logging.Logger,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rl == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := getClientIP(r)
			key := scope + ":ip:" + ip

			allowed, info, err := rl.Allow(r.Context(), key)
			if err != nil {
				logger.Error(
					"rate limiter error",
					"scope", scope,
					"ip", ip,
					"error", err,
				)

				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.Reset.Unix(), 10))

			if !allowed {
				retryAfter := max(int64(time.Until(info.Reset).Seconds()), 0)
				w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))

				logger.Warn(
					"rate limit exceeded",
					"scope", scope,
					"ip", ip,
					"reset", info.Reset,
				)

				httpx.TooManyRequests(w, r, codeRateLimitExceeded, ErrRateLimitExceeded.Error())
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
