package session

import "time"

// Message represents a single conversation turn.
type Message struct {
	Role       string    `json:"role"`        // "user" or "assistant"
	Content    string    `json:"content"`
	TokenCount int       `json:"token_count"` // Estimated tokens
	Timestamp  time.Time `json:"timestamp"`
}

// SessionData represents all serializable session state.
// This data is persisted to Redis and can be restored on service failure.
// It contains conversation history, LLM configuration, STT settings, and tenant configuration.
//
// PERSISTED TO REDIS:
// - ID: unique session identifier
// - CreatedAt, UpdatedAt: timestamps
// - Version: for optimistic locking in distributed deployments
// - ConversationHistory: all user/assistant messages with token counts
// - SystemPrompt: LLM system prompt (from tenant/assistant config)
// - Keyterms: STT keyterm prompting terms (from tenant config)
// - Language, TTSEnabled: feature flags (from tenant/assistant config)
// - AllowedOrigins, RateLimits, Config: tenant settings
type SessionData struct {
	ID                  string         `json:"id"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	Version             int64          `json:"version"` // Monotonically increasing for optimistic locking
	ConversationHistory []Message      `json:"conversation_history"`
	SystemPrompt        string         `json:"system_prompt"`        // LLM system prompt (from tenant/assistant)
	Keyterms            []string       `json:"keyterms"`             // STT keyterm prompting (from tenant config)
	Language            string         `json:"language"`             // Language setting (from tenant/assistant)
	TTSEnabled          bool           `json:"tts_enabled"`          // TTS feature flag (from tenant/assistant)
	AllowedOrigins      []string       `json:"allowed_origins"`      // CORS allowed origins (from tenant)
	RateLimits          map[string]any `json:"rate_limits"`          // Rate limiting config (from tenant)
	Config              map[string]any `json:"config"`               // Additional tenant config
}
