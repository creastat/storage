package drivers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/creastat/session"
	"github.com/redis/go-redis/v9"
)

const (
	// Redis key prefix for sessions
	sessionKeyPrefix = "session:"
	// Default TTL for session keys (24 hours)
	defaultTTL = 24 * time.Hour
)

// RedisStore implements SessionStore using Redis with optimistic locking.
type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisStore creates a new Redis-based session store.
func NewRedisStore(client *redis.Client, ttl time.Duration) *RedisStore {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	return &RedisStore{
		client: client,
		ttl:    ttl,
	}
}

// Create implements SessionStore.
// Creates a new session with Version set to 1 and sets TTL.
func (s *RedisStore) Create(ctx context.Context, data *session.SessionData) error {
	key := s.key(data.ID)
	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now
	data.Version = 1

	val, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, val, s.ttl).Err()
}

// Get implements SessionStore.
// Returns nil if the session is not found (not an error).
// Refreshes TTL on every read.
func (s *RedisStore) Get(ctx context.Context, id string) (*session.SessionData, error) {
	key := s.key(id)
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}

	var data session.SessionData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, err
	}

	// Refresh TTL on read
	if err := s.client.Expire(ctx, key, s.ttl).Err(); err != nil {
		// Log but don't fail if TTL refresh fails
		_ = err
	}

	return &data, nil
}

// Update implements SessionStore.
// Implements optimistic locking using Redis WATCH/MULTI/EXEC.
// Verifies Version matches, increments it, updates UpdatedAt, and persists.
// Returns ErrVersionConflict if the version does not match.
// Returns ErrNotFound if the session does not exist.
// Refreshes TTL on every write.
func (s *RedisStore) Update(ctx context.Context, data *session.SessionData) error {
	key := s.key(data.ID)

	// Use WATCH/MULTI/EXEC for optimistic locking
	err := s.client.Watch(ctx, func(tx *redis.Tx) error {
		// Get current value
		val, err := tx.Get(ctx, key).Result()
		if err == redis.Nil {
			return session.ErrNotFound
		}
		if err != nil {
			return err
		}

		// Unmarshal to check version
		var stored session.SessionData
		if err := json.Unmarshal([]byte(val), &stored); err != nil {
			return err
		}

		// Check version for optimistic locking
		if stored.Version != data.Version {
			return session.ErrVersionConflict
		}

		// Increment version and update timestamp
		data.Version++
		data.UpdatedAt = time.Now()

		// Marshal updated data
		newVal, err := json.Marshal(data)
		if err != nil {
			return err
		}

		// Execute transaction
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newVal, s.ttl)
			return nil
		})
		return err
	}, key)

	return err
}

// Delete implements SessionStore.
func (s *RedisStore) Delete(ctx context.Context, id string) error {
	key := s.key(id)
	return s.client.Del(ctx, key).Err()
}

// Close implements SessionStore.
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// key constructs the Redis key for a session ID.
func (s *RedisStore) key(id string) string {
	return sessionKeyPrefix + id
}
