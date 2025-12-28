package storage

import "errors"

// Common errors for session store operations.
var (
	ErrInvalidConfig    = errors.New("invalid configuration")
	ErrInvalidStoreType = errors.New("invalid store type")
	ErrVersionConflict  = errors.New("session version conflict")
	ErrNotFound         = errors.New("session not found")
)
