package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/infrastructure/config"
	"github.com/fifawcp/api/internal/infrastructure/crypto"
	"github.com/redis/go-redis/v9"
)

// RefreshReplayStorage caches the token pair issued for a just-rotated refresh token,
// keyed by the old token's hash, so concurrent refreshes converge on one successor. The
// payload is encrypted so raw tokens never sit in Redis in plaintext.
type RefreshReplayStorage struct {
	redis  *redis.Client
	cfg    *config.Config
	cipher *crypto.Cipher
}

func NewRefreshReplayStorage(
	redis *redis.Client,
	cfg *config.Config,
	cipher *crypto.Cipher,
) *RefreshReplayStorage {
	return &RefreshReplayStorage{
		redis:  redis,
		cfg:    cfg,
		cipher: cipher,
	}
}

func (s *RefreshReplayStorage) Claim(
	ctx context.Context,
	oldTokenHash string,
	tokens *domain.IssuedTokens,
	ttl time.Duration,
) (bool, *domain.IssuedTokens, error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Redis.QueryTimeout)
	defer cancel()

	payload, err := json.Marshal(tokens)
	if err != nil {
		return false, nil, err
	}

	ciphertext, err := s.cipher.Encrypt(payload)
	if err != nil {
		return false, nil, err
	}

	// SET NX GET claims the slot and returns any successor a concurrent refresh already
	// cached, atomically in one round trip (Redis 7.0+). redis.Nil means we won.
	prev, err := s.redis.SetArgs(ctx, s.createKey(oldTokenHash), ciphertext, redis.SetArgs{
		Mode: "NX",
		TTL:  ttl,
		Get:  true,
	}).Result()
	if err == redis.Nil {
		return true, nil, nil
	}
	if err != nil {
		return false, nil, err
	}

	plaintext, err := s.cipher.Decrypt(prev)
	if err != nil {
		return false, nil, err
	}

	var existing domain.IssuedTokens
	if err := json.Unmarshal(plaintext, &existing); err != nil {
		return false, nil, err
	}

	return false, &existing, nil
}

func (s *RefreshReplayStorage) Release(ctx context.Context, oldTokenHash string) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Redis.QueryTimeout)
	defer cancel()

	return s.redis.Del(ctx, s.createKey(oldTokenHash)).Err()
}

func (s *RefreshReplayStorage) createKey(oldTokenHash string) string {
	return fmt.Sprintf("refresh:replay:%s", oldTokenHash)
}
