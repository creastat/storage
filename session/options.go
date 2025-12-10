package session

import (
	"time"

	"github.com/redis/go-redis/v9"
)

// StoreOption is a functional option for configuring a session store.
type StoreOption func(*storeConfig)

// storeConfig holds configuration for session stores.
type storeConfig struct {
	redisClient *redis.Client
	redisTTL    time.Duration
}

// WithRedisClient sets the Redis client for the Redis store.
func WithRedisClient(client *redis.Client) StoreOption {
	return func(c *storeConfig) {
		c.redisClient = client
	}
}

// WithRedisTTL sets the TTL for Redis keys.
func WithRedisTTL(ttl time.Duration) StoreOption {
	return func(c *storeConfig) {
		c.redisTTL = ttl
	}
}
