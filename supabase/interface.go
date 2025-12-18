package supabase

import (
	"context"
	"time"
)

// Store provides access to Supabase data for chat service operations
type Store interface {
	// GetAssistantByToken retrieves an assistant by its public token
	GetAssistantByToken(ctx context.Context, publicToken string) (*Assistant, error)

	// GetTenant retrieves a tenant by ID
	GetTenant(ctx context.Context, tenantID string) (*Tenant, error)

	// GetSource retrieves a source by ID
	GetSource(ctx context.Context, sourceID string) (*Source, error)

	// GetSourcesByAssistantID retrieves all active sources for an assistant
	GetSourcesByAssistantID(ctx context.Context, assistantID string) ([]Source, error)

	// GetDocument retrieves a document by ID
	GetDocument(ctx context.Context, documentID string) (*Document, error)

	// GetDocumentsByIDs retrieves multiple documents by their IDs
	GetDocumentsByIDs(ctx context.Context, documentIDs []string) ([]Document, error)

	// Close closes the Supabase client and releases resources
	Close() error
}

// Assistant represents an AI assistant from the database
type Assistant struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	Name           string         `json:"name"`
	PublicToken    string         `json:"public_token"`
	SystemPrompt   string         `json:"system_prompt"`
	AllowedOrigins []string       `json:"allowed_origins"`
	RateLimits     map[string]any `json:"rate_limits"`
	Config         map[string]any `json:"config"`
	IsActive       bool           `json:"is_active"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Tenant represents a tenant from the database
type Tenant struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	UserID      string     `json:"user_id"`
	IsTemporary bool       `json:"is_temporary"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Source represents a source from the database
type Source struct {
	ID              string         `json:"id"`
	AssistantID     string         `json:"assistant_id"`
	Name            string         `json:"name"`
	SourceType      string         `json:"source_type"`
	Description     string         `json:"description"`
	Status          string         `json:"status"`
	LastProcessedAt *time.Time     `json:"last_processed_at,omitempty"`
	LastError       string         `json:"last_error"`
	TotalDocuments  int            `json:"total_documents"`
	TotalChunks     int            `json:"total_chunks"`
	Config          map[string]any `json:"config"`
	Metadata        map[string]any `json:"metadata"`
	IsActive        bool           `json:"is_active"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// Document represents a document from the database
type Document struct {
	ID                   string         `json:"id"`
	SourceID             string         `json:"source_id"`
	Title                string         `json:"title"`
	Content              string         `json:"content"`
	URL                  string         `json:"url"`
	Metadata             map[string]any `json:"metadata"`
	Hash                 string         `json:"hash"`
	Language             string         `json:"language"`
	ChunkCount           int            `json:"chunk_count"`
	NormalizedTitle      string         `json:"normalized_title"`
	PageSummary          string         `json:"page_summary"`
	PageSummaryLang      string         `json:"page_summary_lang"`
	ContentHash          string         `json:"content_hash"`
	PrimaryImage         string         `json:"primary_image"`
	CanonicalJSON        map[string]any `json:"canonical_json"`
	ExtractionConfidence float64        `json:"extraction_confidence"`
	PublishedAt          *time.Time     `json:"published_at,omitempty"`
	IsDeleted            bool           `json:"is_deleted"`
	ProcessingStatus     string         `json:"processing_status"`
	LastCrawledAt        time.Time      `json:"last_crawled_at"`
	DetectedLanguage     string         `json:"detected_language"`
	DetectedType         string         `json:"detected_type"`
	ExtractedData        map[string]any `json:"extracted_data"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}
