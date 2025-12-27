package drivers

import (
	"context"
	"sync"
	"time"

	"github.com/creastat/storage/session"
)

// InMemoryStore implements SessionStore using an in-memory map with optimistic locking.
type InMemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*session.SessionData
}

// NewInMemoryStore creates a new in-memory session store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[string]*session.SessionData),
	}
}

// Create implements SessionStore.
// Creates a new session with Version set to 1.
func (s *InMemoryStore) Create(ctx context.Context, data *session.SessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	data.CreatedAt = now
	data.UpdatedAt = now
	data.Version = 1

	s.sessions[data.ID] = data
	return nil
}

// Get implements SessionStore.
// Returns nil if the session is not found (not an error).
func (s *InMemoryStore) Get(ctx context.Context, id string) (*session.SessionData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, exists := s.sessions[id]
	if !exists {
		return nil, nil // Not found
	}
	return data, nil
}

// Update implements SessionStore.
// Implements optimistic locking: verifies Version matches, increments it,
// updates UpdatedAt, and persists the SessionData.
// Returns ErrVersionConflict if the version does not match.
// Returns ErrNotFound if the session does not exist.
func (s *InMemoryStore) Update(ctx context.Context, data *session.SessionData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, exists := s.sessions[data.ID]
	if !exists {
		return session.ErrNotFound
	}

	// Check version for optimistic locking
	if stored.Version != data.Version {
		return session.ErrVersionConflict
	}

	// Increment version and update timestamp
	data.Version++
	data.UpdatedAt = time.Now()

	s.sessions[data.ID] = data
	return nil
}

// Delete implements SessionStore.
func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, id)
	return nil
}

// Close implements SessionStore.
func (s *InMemoryStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = nil
	return nil
}
