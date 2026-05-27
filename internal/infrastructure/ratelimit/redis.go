package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	client *redis.Client
	cfg    config.RateLimitTier
}

func NewRedisRateLimiter(client *redis.Client, cfg config.RateLimitTier) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		cfg:    cfg,
	}
}

func (rl *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, *LimitInfo, error) {
	// Divide time into fixed windows. All requests within the same window share one counter.
	// Example window size: 1 hour → bucket changes every 3600 seconds.
	windowSeconds := int64(rl.cfg.Window.Seconds())
	// Bucket is the window start time (Unix timestamp)
	bucket := time.Now().Unix() / windowSeconds

	// Full Redis key: caller provides <scope + ip>, we append the window bucket.
	// Example: "ratelimit:otp_request:ip:203.0.113.42:1744578"
	windowKey := fmt.Sprintf("ratelimit:%s:%d", key, bucket)

	// Increment counter
	count, err := rl.client.Incr(ctx, windowKey).Result()
	if err != nil {
		return false, nil, err
	}

	// Set TTL only on first increment so stale keys auto-expire.
	// A small race between INCR and EXPIRE is harmless — worst case the
	// key persists one extra window before Redis cleans it up.
	if count == 1 {
		rl.client.Expire(ctx, windowKey, rl.cfg.Window)
	}

	// Calculate metadata
	remaining := max(rl.cfg.RequestsPerWindow-int(count), 0)

	reset := time.Unix((bucket+1)*windowSeconds, 0)
	allowed := count <= int64(rl.cfg.RequestsPerWindow)

	info := &LimitInfo{
		Limit:      rl.cfg.RequestsPerWindow,
		Remaining:  remaining,
		Reset:      reset,
		RetryAfter: rl.cfg.Window,
	}

	return allowed, info, nil
}
