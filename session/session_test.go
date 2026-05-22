package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// TestNewStoreMemory creates an in-memory store.
func TestNewStoreMemory(t *testing.T) {
	store, err := NewStore(StoreTypeMemory)
	require.NoError(t, err)
	require.NotNil(t, store)
}

// TestNewStoreInvalidType rejects unknown store type.
func TestNewStoreInvalidType(t *testing.T) {
	store, err := NewStore(StoreType("unknown"))
	require.Error(t, err)
	require.Nil(t, store)
	require.True(t, errors.Is(err, ErrInvalidStoreType))
}

// TestNewStoreRedisNoClient requires Redis client.
func TestNewStoreRedisNoClient(t *testing.T) {
	store, err := NewStore(StoreTypeRedis)
	require.Error(t, err)
	require.Nil(t, store)
	require.True(t, errors.Is(err, ErrInvalidConfig))
}

// TestNewStoreRedisWithClient creates a Redis store.
func TestNewStoreRedisWithClient(t *testing.T) {
	mr := miniredis.NewMiniRedis()
	require.NoError(t, mr.Start())
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store, err := NewStore(StoreTypeRedis, WithRedisClient(client))
	require.NoError(t, err)
	require.NotNil(t, store)
	require.NoError(t, store.Close())
}

// TestInMemoryStoreCreate adds a new session.
func TestInMemoryStoreCreate(t *testing.T) {
	store, err := NewStore(StoreTypeMemory)
	require.NoError(t, err)

	session := &SessionData{
		ID: "sess-123",
	}

	err = store.Create(context.Background(), session)
	require.NoError(t, err)
	require.NotZero(t, session.CreatedAt)
	require.NotZero(t, session.UpdatedAt)
	require.Equal(t, int64(1), session.Version)
}

// TestInMemoryStoreGet retrieves a session.
func TestInMemoryStoreGet(t *testing.T) {
	store, err := NewStore(StoreTypeMemory)
	require.NoError(t, err)

	session := &SessionData{ID: "sess-123"}
	err = store.Create(context.Background(), session)
	require.NoError(t, err)

	retrieved, err := store.Get(context.Background(), "sess-123")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, "sess-123", retrieved.ID)
	require.Equal(t, int64(1), retrieved.Version)
}

// TestInMemoryStoreGetNotFound returns nil for missing session.
func TestInMemoryStoreGetNotFound(t *testing.T) {
	store, err := NewStore(StoreTypeMemory)
	require.NoError(t, err)

	retrieved, err := store.Get(context.Background(), "nonexistent")
	require.NoError(t, err)
	require.Nil(t, retrieved)
}

// TestInMemoryStoreUpdate modifies an existing session.
func TestInMemoryStoreUpdate(t *testing.T) {
	store, err := NewStore(StoreTypeMemory)
	require.NoError(t, err)

	session := &SessionData{ID: "sess-123"}
	err = store.Create(context.Background(), session)
	require.NoError(t, err)

	createdAt := session.CreatedAt
	session.ConversationHistory = append(session.ConversationHistory, Message{
		Role:    "user",
		Content: "hello",
	})

	err = store.Update(context.Background(), session)
	require.NoError(t, err)
	require.Equal(t, int64(2), session.Version)
	require.True(t, session.UpdatedAt.After(createdAt))
}

// TestInMemoryStoreUpdateVersionConflict rejects stale updates.
func TestInMemoryStoreUpdateVersionConflict(t *testing.T) {
	store, err := NewStore(StoreTypeMemory)
	require.NoError(t, err)

	session := &SessionData{ID: "sess-123"}
	store.Create(context.Background(), session)

	// Fetch and update once to bump version
	retrieved, _ := store.Get(context.Background(), "sess-123")
	store.Update(context.Background(), retrieved)

	// Try to update with stale version using a detached snapshot.
	stale := &SessionData{ID: "sess-123", Version: 1}
	err = store.Update(context.Background(), stale)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrVersionConflict))
}

// TestInMemoryStoreUpdateNotFound returns error for missing session.
func TestInMemoryStoreUpdateNotFound(t *testing.T) {
	store, err := NewStore(StoreTypeMemory)
	require.NoError(t, err)

	session := &SessionData{ID: "nonexistent", Version: 1}
	err = store.Update(context.Background(), session)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound))
}

// TestInMemoryStoreDelete removes a session.
func TestInMemoryStoreDelete(t *testing.T) {
	store, err := NewStore(StoreTypeMemory)
	require.NoError(t, err)

	session := &SessionData{ID: "sess-123"}
	store.Create(context.Background(), session)

	err = store.Delete(context.Background(), "sess-123")
	require.NoError(t, err)

	retrieved, _ := store.Get(context.Background(), "sess-123")
	require.Nil(t, retrieved)
}

// TestInMemoryStoreClose closes the store.
func TestInMemoryStoreClose(t *testing.T) {
	store, err := NewStore(StoreTypeMemory)
	require.NoError(t, err)

	err = store.Close()
	require.NoError(t, err)
}

// TestRedisStoreCreate adds a session to Redis.
func TestRedisStoreCreate(t *testing.T) {
	mr := miniredis.NewMiniRedis()
	require.NoError(t, mr.Start())
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store, err := NewStore(StoreTypeRedis, WithRedisClient(client))
	require.NoError(t, err)

	session := &SessionData{ID: "sess-123"}
	err = store.Create(context.Background(), session)
	require.NoError(t, err)
	require.NotZero(t, session.CreatedAt)

	store.Close()
}

// TestRedisStoreGet retrieves a session from Redis.
func TestRedisStoreGet(t *testing.T) {
	mr := miniredis.NewMiniRedis()
	require.NoError(t, mr.Start())
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store, err := NewStore(StoreTypeRedis, WithRedisClient(client))
	require.NoError(t, err)

	session := &SessionData{
		ID:       "sess-123",
		Language: "en",
	}
	store.Create(context.Background(), session)

	retrieved, err := store.Get(context.Background(), "sess-123")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, "sess-123", retrieved.ID)
	require.Equal(t, "en", retrieved.Language)

	store.Close()
}

// TestRedisStoreGetNotFound returns nil for missing session.
func TestRedisStoreGetNotFound(t *testing.T) {
	mr := miniredis.NewMiniRedis()
	require.NoError(t, mr.Start())
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store, err := NewStore(StoreTypeRedis, WithRedisClient(client))
	require.NoError(t, err)

	retrieved, err := store.Get(context.Background(), "nonexistent")
	require.NoError(t, err)
	require.Nil(t, retrieved)

	store.Close()
}

// TestRedisStoreUpdate modifies a session.
func TestRedisStoreUpdate(t *testing.T) {
	mr := miniredis.NewMiniRedis()
	require.NoError(t, mr.Start())
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store, err := NewStore(StoreTypeRedis, WithRedisClient(client))
	require.NoError(t, err)

	session := &SessionData{ID: "sess-123"}
	store.Create(context.Background(), session)

	retrieved, _ := store.Get(context.Background(), "sess-123")
	retrieved.Language = "es"
	err = store.Update(context.Background(), retrieved)
	require.NoError(t, err)

	updated, _ := store.Get(context.Background(), "sess-123")
	require.Equal(t, "es", updated.Language)

	store.Close()
}

// TestRedisStoreDelete removes a session.
func TestRedisStoreDelete(t *testing.T) {
	mr := miniredis.NewMiniRedis()
	require.NoError(t, mr.Start())
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store, err := NewStore(StoreTypeRedis, WithRedisClient(client))
	require.NoError(t, err)

	session := &SessionData{ID: "sess-123"}
	store.Create(context.Background(), session)

	err = store.Delete(context.Background(), "sess-123")
	require.NoError(t, err)

	retrieved, _ := store.Get(context.Background(), "sess-123")
	require.Nil(t, retrieved)

	store.Close()
}

// TestSessionDataStruct tests the SessionData structure.
func TestSessionDataStruct(t *testing.T) {
	session := &SessionData{
		ID:             "sess-123",
		Version:        1,
		Language:       "en",
		TTSEnabled:     true,
		AllowedOrigins: []string{"https://example.com"},
		ConversationHistory: []Message{
			{Role: "user", Content: "hello"},
		},
	}

	require.Equal(t, "sess-123", session.ID)
	require.Equal(t, int64(1), session.Version)
	require.Equal(t, "en", session.Language)
	require.True(t, session.TTSEnabled)
	require.Equal(t, 1, len(session.AllowedOrigins))
	require.Equal(t, 1, len(session.ConversationHistory))
}

// TestMessageStruct tests the Message structure.
func TestMessageStruct(t *testing.T) {
	msg := Message{
		Role:       "assistant",
		Content:    "How can I help?",
		TokenCount: 5,
		Timestamp:  time.Now(),
	}

	require.Equal(t, "assistant", msg.Role)
	require.Equal(t, "How can I help?", msg.Content)
	require.Equal(t, 5, msg.TokenCount)
	require.NotZero(t, msg.Timestamp)
}

// TestSessionDataConversationHistory tests conversation history management.
func TestSessionDataConversationHistory(t *testing.T) {
	session := &SessionData{
		ID: "sess-123",
		ConversationHistory: []Message{
			{Role: "user", Content: "what is X?"},
			{Role: "assistant", Content: "X is ..."},
			{Role: "user", Content: "how do I use X?"},
			{Role: "assistant", Content: "To use X, ..."},
		},
	}

	require.Equal(t, 4, len(session.ConversationHistory))
	require.Equal(t, "user", session.ConversationHistory[0].Role)
	require.Equal(t, "assistant", session.ConversationHistory[1].Role)
}

// TestSessionDataMetadata tests metadata fields.
func TestSessionDataMetadata(t *testing.T) {
	now := time.Now()
	session := &SessionData{
		ID:        "sess-123",
		CreatedAt: now,
		UpdatedAt: now.Add(time.Hour),
		Version:   5,
	}

	require.Equal(t, now, session.CreatedAt)
	require.True(t, session.UpdatedAt.After(session.CreatedAt))
	require.Equal(t, int64(5), session.Version)
}

// TestSessionDataConfig tests configuration maps.
func TestSessionDataConfig(t *testing.T) {
	session := &SessionData{
		ID: "sess-123",
		RateLimits: map[string]any{
			"requests_per_minute": 60,
			"messages_per_hour":   100,
		},
		Config: map[string]any{
			"feature_flag_1": true,
			"feature_flag_2": false,
		},
	}

	require.Equal(t, 60, session.RateLimits["requests_per_minute"])
	require.Equal(t, true, session.Config["feature_flag_1"])
}
