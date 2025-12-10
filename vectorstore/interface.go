package vectorstore

import "context"

// VectorStore is a technology-agnostic interface for vector similarity search.
// Implementations can use Qdrant, Pinecone, Supabase Vector, Weaviate, etc.
type VectorStore interface {
	// Search performs vector similarity search with optional filtering.
	Search(ctx context.Context, vector []float32, filter SearchFilter, limit int) ([]SearchResult, error)

	// Close releases any resources held by the vector store.
	Close() error
}

// SearchFilter defines filtering options for vector search.
type SearchFilter struct {
	// SourceID filters results to a specific source/collection.
	SourceID string

	// Metadata filters results by metadata key-value pairs.
	Metadata map[string]any

	// MinScore filters results below this similarity threshold (0.0-1.0).
	MinScore float32
}

// SearchResult represents a single result from vector similarity search.
type SearchResult struct {
	// ID is the unique identifier of the result.
	ID string

	// Score is the similarity score (0.0-1.0, higher is more similar).
	Score float32

	// Content is the text content associated with this vector.
	Content string

	// SourceID identifies the source/collection this result belongs to.
	SourceID string

	// DocumentID identifies the document this chunk belongs to.
	DocumentID string

	// Metadata contains additional key-value pairs.
	Metadata map[string]any
}
