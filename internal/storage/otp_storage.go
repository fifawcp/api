package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

type OTPStorage struct {
	redis *redis.Client
	cfg   *config.Config
}

func NewOTPStorage(
	redis *redis.Client,
	cfg *config.Config,
) *OTPStorage {
	return &OTPStorage{
		redis: redis,
		cfg:   cfg,
	}
}

func (r *OTPStorage) SetOTP(
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

func (r *OTPStorage) GetOTP(
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

func (r *OTPStorage) IncrementAttempts(
	ctx context.Context,
	identifier string,
	purpose domain.OTPPurpose,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.Redis.QueryTimeout)
	defer cancel()

	key := r.createKey(purpose, identifier)

	otp, err := r.GetOTP(ctx, identifier, purpose)
	if err != nil {
		return err
	}

	otp.Attempts++

	// Update in Redis (preserve TTL)
	ttl, _ := r.redis.TTL(ctx, key).Result()
	return r.SetOTP(ctx, otp, ttl)
}

func (r *OTPStorage) DeleteOTP(
	ctx context.Context,
	identifier string,
	purpose domain.OTPPurpose,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.cfg.Redis.QueryTimeout)
	defer cancel()

	key := r.createKey(purpose, identifier)

	return r.redis.Del(ctx, key).Err()
}

func (r *OTPStorage) createKey(purpose domain.OTPPurpose, identifier string) string {
	return fmt.Sprintf("otp:%s:%s", purpose, identifier)
}
