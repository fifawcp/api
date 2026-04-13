package storage

import (
	"context"
	"encoding/json"

	"github.com/ncondes/fifawcp/internal/domain"
	"github.com/ncondes/fifawcp/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

type UserStorage struct {
	redis *redis.Client
	cfg   *config.Config
}

func NewUserStorage(
	redis *redis.Client,
	cfg *config.Config,
) *UserStorage {
	return &UserStorage{
		redis: redis,
		cfg:   cfg,
	}
}

func (s *UserStorage) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Redis.QueryTimeout)
	defer cancel()

	key := s.createKey(userID)

	data, err := s.redis.Get(ctx, key).Result()
	switch {
	case err == redis.Nil:
		return nil, nil // Cache miss - user not found in cache (not an error)
	case err != nil:
		return nil, err
	}

	var user domain.User

	if err := json.Unmarshal([]byte(data), &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *UserStorage) SetUser(ctx context.Context, user *domain.User) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Redis.QueryTimeout)
	defer cancel()

	key := s.createKey(user.ID)

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, data, s.cfg.Redis.UserCacheTTL).Err()
}

func (s *UserStorage) createKey(userID string) string {
	return "user:" + userID
}
