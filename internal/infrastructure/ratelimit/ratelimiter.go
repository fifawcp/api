package ratelimit

import (
	"context"
	"time"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, *LimitInfo, error)
}

type LimitInfo struct {
	Limit      int           // Maximum requests allowed
	Remaining  int           // Remaining requests in window
	Reset      time.Time     // When the limit resets
	RetryAfter time.Duration // Time to wait before retrying
}
