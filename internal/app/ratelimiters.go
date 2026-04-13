package app

import (
	"github.com/ncondes/fifawcp/internal/infrastructure/config"
	"github.com/ncondes/fifawcp/internal/infrastructure/ratelimit"
	"github.com/redis/go-redis/v9"
)

type RateLimiters struct {
	StrictIP   ratelimit.RateLimiter
	ModerateIP ratelimit.RateLimiter
	RelaxedIP  ratelimit.RateLimiter
}

func newRateLimiters(
	rc *redis.Client,
	cfg *config.RateLimitConfig,
) *RateLimiters {
	// If rate limiting is disabled or Redis client is nil, return empty rate limiters
	if !cfg.Enabled || rc == nil {
		return &RateLimiters{}
	}

	return &RateLimiters{
		StrictIP:   ratelimit.NewRedisRateLimiter(rc, cfg.StrictIP),
		ModerateIP: ratelimit.NewRedisRateLimiter(rc, cfg.ModerateIP),
		RelaxedIP:  ratelimit.NewRedisRateLimiter(rc, cfg.RelaxedIP),
	}
}
