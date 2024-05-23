package database

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/likimiad/ozon_fintech/internal/config"
)

var ErrRedisConnect = errors.New("failed to connect to redis")
var ctx = context.Background()

// NewRedisClient creates a new Redis client and verifies the connection.
func NewRedisClient(cfg config.RedisConfig) (*redis.Client, error) {
	rc := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	_, err := rc.Ping(ctx).Result()
	if err != nil {
		return nil, ErrRedisConnect
	}

	return rc, nil
}
