# Session Storage Library

This library provides an abstraction for managing chat sessions with support for multiple storage backends.

## Features

- **In-Memory Store**: Fast, local storage for single-instance deployments.
- **Redis Store**: Distributed storage for multi-instance deployments.
- **Serializable Session Data**: JSON-serializable session data for persistence.
- **Extensible**: Easy to add new storage backends.

## Usage

```go
import (
    "context"
    "github.com/creastat/storage/session"
    "github.com/creastat/storage/session/drivers"
    "github.com/go-redis/redis/v8"
)

// Create an in-memory store
store, err := session.NewStore(session.StoreTypeMemory)
if err != nil {
    // handle error
}

// Create a Redis store
rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})
store, err = session.NewStore(session.StoreTypeRedis, session.WithRedisClient(rdb))
if err != nil {
    // handle error
}

// Use the store
ctx := context.Background()
data := &session.SessionData{
    ID:          "session-123",
    AssistantID: "assistant-456",
    TenantID:    "tenant-789",
    TTSEnabled:  true,
    Language:    "en",
}
err = store.Create(ctx, data)
if err != nil {
    // handle error
}

// Retrieve session
retrieved, err := store.Get(ctx, "session-123")
if err != nil {
    // handle error
}
```

## Session Data

The `SessionData` struct contains serializable fields for a chat session:

- `ID`: Unique session identifier
- `AssistantID`: Associated assistant ID
- `TenantID`: Associated tenant ID
- `CreatedAt`: Creation timestamp
- `UpdatedAt`: Last update timestamp
- `TTSEnabled`: Text-to-speech enabled flag
- `Language`: Session language code

## Drivers

### In-Memory

- Fast, local storage
- Data lost on restart
- Suitable for single-instance deployments

### Redis

- Distributed storage
- Data persists across restarts
- Suitable for multi-instance deployments
- Configurable TTL (default: 24 hours)

## Extending

To add a new storage backend:

1. Implement the `Store` interface
2. Add a new `StoreType` constant
3. Update the `NewStore` factory function
4. Add any necessary options
