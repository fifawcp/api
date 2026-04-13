package cache

import (
	"context"
	"time"

	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(
	cfg *config.Config,
) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: cfg.Redis.PoolSize,
	})

	// Create a context with a timeout to limit the time we wait for the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping the database to verify the connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
