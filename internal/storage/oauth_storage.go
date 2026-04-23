package storage

import (
	"context"
	"fmt"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

type OAuthStorage struct {
	redis *redis.Client
	cfg   *config.Config
}

func NewOAuthStorage(
	redis *redis.Client,
	cfg *config.Config,
) *OAuthStorage {
	return &OAuthStorage{
		redis: redis,
		cfg:   cfg,
	}
}

func (s *OAuthStorage) SetOAuthState(ctx context.Context, state string, payload string) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Redis.QueryTimeout)
	defer cancel()

	key := s.createKey(state)

	return s.redis.Set(ctx, key, payload, s.cfg.Auth.GoogleOAuth.StateTTL).Err()
}

func (s *OAuthStorage) GetAndDeleteOAuthState(ctx context.Context, state string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Redis.QueryTimeout)
	defer cancel()

	key := s.createKey(state)

	data, err := s.redis.GetDel(ctx, key).Result()
	switch {
	case err == redis.Nil:
		return "", domain.ErrOAuthStateNotFound
	case err != nil:
		return "", err
	}

	return data, nil
}

func (s *OAuthStorage) createKey(state string) string {
	return fmt.Sprintf("oauth:state:%s", state)
}
