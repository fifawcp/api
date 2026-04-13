package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ncondes/fifa-world-cup-pickems/internal/domain"
	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

type OTPRepository struct {
	redis *redis.Client
	cfg   *config.Config
}

func NewOTPRepository(
	redis *redis.Client,
	cfg *config.Config,
) *OTPRepository {
	return &OTPRepository{
		redis: redis,
		cfg:   cfg,
	}
}

func (r *OTPRepository) SetOTP(
	ctx context.Context,
	otp *domain.OTP,
	ttl time.Duration,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.Redis.QueryTimeout)
	defer cancel()

	key := r.createKey(otp.Purpose, otp.Identifier)

	jsonData, err := json.Marshal(otp)
	if err != nil {
		return err
	}

	return r.redis.Set(ctx, key, jsonData, ttl).Err()
}

func (r *OTPRepository) GetOTP(
	ctx context.Context,
	identifier string,
	purpose domain.OTPPurpose,
) (*domain.OTP, error) {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.Redis.QueryTimeout)
	defer cancel()

	key := r.createKey(purpose, identifier)

	data, err := r.redis.Get(ctx, key).Result()
	switch {
	case err == redis.Nil:
		return nil, domain.ErrOTPInvalidOrExpired
	case err != nil:
		return nil, err
	}

	var otp domain.OTP

	if err := json.Unmarshal([]byte(data), &otp); err != nil {
		return nil, err
	}

	return &otp, nil
}

func (r *OTPRepository) IncrementAttempts(
	ctx context.Context,
	identifier string,
	purpose domain.OTPPurpose,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.Redis.QueryTimeout)
	defer cancel()

	key := r.createKey(purpose, identifier)

	// Get current data
	otp, err := r.GetOTP(ctx, identifier, purpose)
	if err != nil {
		return err
	}

	// Increment attempts
	otp.Attempts++

	// Update in Redis (preserve TTL)
	ttl, _ := r.redis.TTL(ctx, key).Result()
	return r.SetOTP(ctx, otp, ttl)
}

func (r *OTPRepository) DeleteOTP(
	ctx context.Context,
	identifier string,
	purpose domain.OTPPurpose,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.Redis.QueryTimeout)
	defer cancel()

	key := r.createKey(purpose, identifier)
	return r.redis.Del(ctx, key).Err()
}

func (r *OTPRepository) createKey(purpose domain.OTPPurpose, identifier string) string {
	return fmt.Sprintf("otp:%s:%s", purpose, identifier)
}
