package supabase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/supabase-community/supabase-go"
)

// Config holds Supabase connection configuration
type Config struct {
	URL      string
	APIKey   string
	CacheTTL time.Duration // Default: 5 minutes
}

// Client implements the Store interface using Supabase
type Client struct {
	client   *supabase.Client
	cache    *cache
	cacheTTL time.Duration
}

// cache provides thread-safe caching for frequently accessed data
type cache struct {
	mu        sync.RWMutex
	byToken   map[string]*cacheEntry[*Assistant]
	byID      map[string]*cacheEntry[any]
}

type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

// New creates a new Supabase client
func New(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("supabase URL is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("supabase API key is required")
	}

	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 5 * time.Minute
	}

	client, err := supabase.NewClient(cfg.URL, cfg.APIKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create supabase client: %w", err)
	}

	return &Client{
		client:   client,
		cacheTTL: cfg.CacheTTL,
		cache: &cache{
			byToken: make(map[string]*cacheEntry[*Assistant]),
			byID:    make(map[string]*cacheEntry[any]),
		},
	}, nil
}

// GetAssistantByToken retrieves an assistant by its public token
func (c *Client) GetAssistantByToken(ctx context.Context, publicToken string) (*Assistant, error) {
	// Check cache first
	if cached := c.getFromCacheByToken(publicToken); cached != nil {
		return cached, nil
	}

	var assistants []Assistant
	_, err := c.client.From("assistants").
		Select("*", "", false).
		Eq("public_token", publicToken).
		ExecuteTo(&assistants)

	if err != nil {
		return nil, fmt.Errorf("failed to get assistant by token: %w", err)
	}

	if len(assistants) == 0 {
		return nil, fmt.Errorf("assistant not found")
	}

	assistant := &assistants[0]

	// Cache by token and ID
	c.addToCache("token", publicToken, assistant)
	c.addToCache("id", assistant.ID, assistant)

	return assistant, nil
}

// GetTenant retrieves a tenant by ID
func (c *Client) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
	// Check cache first
	if cached, ok := c.getFromCacheByID(tenantID).(*Tenant); ok && cached != nil {
		return cached, nil
	}

	var tenant Tenant
	_, err := c.client.From("tenants").
		Select("*", "", false).
		Eq("id", tenantID).
		Single().
		ExecuteTo(&tenant)

	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Cache by ID
	c.addToCache("id", tenantID, &tenant)

	return &tenant, nil
}

// GetSource retrieves a source by ID
func (c *Client) GetSource(ctx context.Context, sourceID string) (*Source, error) {
	// Check cache first
	if cached, ok := c.getFromCacheByID(sourceID).(*Source); ok && cached != nil {
		return cached, nil
	}

	var source Source
	_, err := c.client.From("sources").
		Select("*", "", false).
		Eq("id", sourceID).
		Single().
		ExecuteTo(&source)

	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	// Cache by ID
	c.addToCache("id", sourceID, &source)

	return &source, nil
}

// GetSourcesByAssistantID retrieves all active sources for an assistant
func (c *Client) GetSourcesByAssistantID(ctx context.Context, assistantID string) ([]Source, error) {
	var sources []Source
	_, err := c.client.From("sources").
		Select("*", "", false).
		Eq("assistant_id", assistantID).
		Eq("is_active", "true").
		ExecuteTo(&sources)

	if err != nil {
		return nil, fmt.Errorf("failed to get sources by assistant_id: %w", err)
	}

	// Cache each source by ID
	for i := range sources {
		c.addToCache("id", sources[i].ID, &sources[i])
	}

	return sources, nil
}

// GetDocument retrieves a document by ID
func (c *Client) GetDocument(ctx context.Context, documentID string) (*Document, error) {
	// Check cache first
	if cached, ok := c.getFromCacheByID(documentID).(*Document); ok && cached != nil {
		return cached, nil
	}

	var document Document
	_, err := c.client.From("documents").
		Select("*", "", false).
		Eq("id", documentID).
		Single().
		ExecuteTo(&document)

	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Cache by ID
	c.addToCache("id", documentID, &document)

	return &document, nil
}

// GetDocumentsByIDs retrieves multiple documents by their IDs
func (c *Client) GetDocumentsByIDs(ctx context.Context, documentIDs []string) ([]Document, error) {
	if len(documentIDs) == 0 {
		return []Document{}, nil
	}

	var documents []Document
	_, err := c.client.From("documents").
		Select("*", "", false).
		In("id", documentIDs).
		ExecuteTo(&documents)

	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	return documents, nil
}

// Close closes the Supabase client
func (c *Client) Close() error {
	// Supabase client doesn't require explicit close
	return nil
}

// getFromCacheByToken retrieves an assistant from cache by token
func (c *Client) getFromCacheByToken(key string) *Assistant {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	if e, ok := c.cache.byToken[key]; ok {
		if time.Now().Before(e.expiresAt) {
			return e.value
		}
	}
	return nil
}

// getFromCacheByID retrieves a value from cache by ID
func (c *Client) getFromCacheByID(key string) any {
	c.cache.mu.RLock()
	defer c.cache.mu.RUnlock()

	if e, ok := c.cache.byID[key]; ok {
		if time.Now().Before(e.expiresAt) {
			return e.value
		}
	}
	return nil
}

// addToCache adds a value to cache
func (c *Client) addToCache(keyType, key string, value any) {
	c.cache.mu.Lock()
	defer c.cache.mu.Unlock()

	entry := &cacheEntry[any]{
		value:     value,
		expiresAt: time.Now().Add(c.cacheTTL),
	}

	switch keyType {
	case "token":
		if assistant, ok := value.(*Assistant); ok {
			c.cache.byToken[key] = &cacheEntry[*Assistant]{
				value:     assistant,
				expiresAt: entry.expiresAt,
			}
		}
	case "id":
		c.cache.byID[key] = entry
	}
}

// Compile-time check that Client implements Store
var _ Store = (*Client)(nil)
