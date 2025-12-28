package session

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// StoreType represents the type of session store.
type StoreType string

const (
	StoreTypeMemory StoreType = "memory"
	StoreTypeRedis  StoreType = "redis"
)

// NewStore creates a new SessionStore based on the given type.
// Supports "memory" and "redis" driver types.
// For Redis, requires WithRedisClient option.
func NewStore(storeType StoreType, opts ...StoreOption) (Store, error) {
	config := &storeConfig{}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	switch storeType {
	case StoreTypeMemory:
		return &inMemoryStore{
			sessions: make(map[string]*SessionData),
		}, nil

	case StoreTypeRedis:
		if config.redisClient == nil {
			return nil, ErrInvalidConfig
		}
		ttl := config.redisTTL
		if ttl <= 0 {
			ttl = 24 * time.Hour
		}
		return &redisStore{
			client: config.redisClient,
			ttl:    ttl,
		}, nil

	default:
		return nil, ErrInvalidStoreType
	}
}

// inMemoryStore implements Store using an in-memory map with optimistic locking.
type inMemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
}

// Create implements Store.
func (s *inMemoryStore) Create(ctx context.Context, data *SessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now
	data.Version = 1

	s.sessions[data.ID] = data
	return nil
}

// Get implements Store.
func (s *inMemoryStore) Get(ctx context.Context, id string) (*SessionData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.sessions[id]
	if !exists {
		return nil, nil
	}
	return data, nil
}

// Update implements Store.
func (s *inMemoryStore) Update(ctx context.Context, data *SessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, exists := s.sessions[data.ID]
	if !exists {
		return ErrNotFound
	}

	if stored.Version != data.Version {
		return ErrVersionConflict
	}

	data.Version++
	data.UpdatedAt = time.Now()

	s.sessions[data.ID] = data
	return nil
}

// Delete implements Store.
func (s *inMemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
	return nil
}

// Close implements Store.
func (s *inMemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = nil
	return nil
}

// redisStore implements Store using Redis with optimistic locking.
type redisStore struct {
	client *redis.Client
	ttl    time.Duration
}

// Create implements Store.
func (s *redisStore) Create(ctx context.Context, data *SessionData) error {
	key := "session:" + data.ID
	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now
	data.Version = 1

	val, err := marshalJSON(data)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, val, s.ttl).Err()
}

// Get implements Store.
func (s *redisStore) Get(ctx context.Context, id string) (*SessionData, error) {
	key := "session:" + id
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var data SessionData
	if err := unmarshalJSON([]byte(val), &data); err != nil {
		return nil, err
	}

	// Refresh TTL on read
	_ = s.client.Expire(ctx, key, s.ttl).Err()

	return &data, nil
}

// Update implements Store.
func (s *redisStore) Update(ctx context.Context, data *SessionData) error {
	key := "session:" + data.ID

	err := s.client.Watch(ctx, func(tx *redis.Tx) error {
		val, err := tx.Get(ctx, key).Result()
		if err == redis.Nil {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		var stored SessionData
		if err := unmarshalJSON([]byte(val), &stored); err != nil {
			return err
		}

		if stored.Version != data.Version {
			return ErrVersionConflict
		}

		data.Version++
		data.UpdatedAt = time.Now()

		newVal, err := marshalJSON(data)
		if err != nil {
			return err
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, key, newVal, s.ttl)
			return nil
		})
		return err
	}, key)

	return err
}

// Delete implements Store.
func (s *redisStore) Delete(ctx context.Context, id string) error {
	key := "session:" + id
	return s.client.Del(ctx, key).Err()
}

// Close implements Store.
func (s *redisStore) Close() error {
	return s.client.Close()
}

// Helper functions for JSON marshaling
func marshalJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	return string(b), err
}

func unmarshalJSON(b []byte, v any) error {
	return json.Unmarshal(b, v)
}
