package session

import "context"

// Store defines the interface for session storage operations.
type Store interface {
	// Create creates a new session with Version set to 1.
	// Returns an error if the session already exists.
	Create(ctx context.Context, data *SessionData) error

	// Get retrieves a session by ID.
	// Returns nil if the session is not found (not an error).
	Get(ctx context.Context, id string) (*SessionData, error)

	// Update updates an existing session with optimistic locking.
	// Verifies the Version matches the stored version, increments Version,
	// updates UpdatedAt timestamp, and persists the SessionData.
	// Returns ErrVersionConflict if the version does not match.
	// Returns ErrNotFound if the session does not exist.
	Update(ctx context.Context, data *SessionData) error

	// Delete deletes a session by ID.
	Delete(ctx context.Context, id string) error

	// Close closes the store and releases any resources.
	Close() error
}


