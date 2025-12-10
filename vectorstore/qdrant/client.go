package qdrant

import (
	"context"
	"fmt"

	"github.com/creastat/storage/vectorstore"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds Qdrant connection configuration.
type Config struct {
	// URL is the Qdrant server address (e.g., "localhost:6334").
	URL string

	// CollectionName is the name of the collection to search.
	CollectionName string

	// APIKey is optional API key for authentication.
	APIKey string
}

// Client implements vectorstore.VectorStore for Qdrant.
type Client struct {
	conn           *grpc.ClientConn
	points         qdrant.PointsClient
	collectionName string
}

// New creates a new Qdrant client.
func New(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("qdrant url is required")
	}

	// TODO: Add support for API Key / TLS
	conn, err := grpc.NewClient(cfg.URL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to qdrant: %w", err)
	}

	return &Client{
		conn:           conn,
		points:         qdrant.NewPointsClient(conn),
		collectionName: cfg.CollectionName,
	}, nil
}

// Search implements vectorstore.VectorStore.
func (c *Client) Search(ctx context.Context, vector []float32, filter vectorstore.SearchFilter, limit int) ([]vectorstore.SearchResult, error) {
	// Build Qdrant filter
	qFilter := buildFilter(filter)

	res, err := c.points.Search(ctx, &qdrant.SearchPoints{
		CollectionName: c.collectionName,
		Vector:         vector,
		Limit:          uint64(limit),
		Filter:         qFilter,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant search failed: %w", err)
	}

	// Convert results
	results := make([]vectorstore.SearchResult, 0, len(res.Result))
	for _, point := range res.Result {
		// Apply min score filter
		if filter.MinScore > 0 && point.Score < filter.MinScore {
			continue
		}

		result := vectorstore.SearchResult{
			Score:    point.Score,
			Metadata: make(map[string]any),
		}

		// Extract ID
		if point.Id != nil {
			if uuid := point.Id.GetUuid(); uuid != "" {
				result.ID = uuid
			} else if num := point.Id.GetNum(); num != 0 {
				result.ID = fmt.Sprintf("%d", num)
			}
		}

		// Extract payload
		if point.Payload != nil {
			for k, v := range point.Payload {
				switch k {
				case "content":
					if str := v.GetStringValue(); str != "" {
						result.Content = str
					}
				case "source_id":
					if str := v.GetStringValue(); str != "" {
						result.SourceID = str
					}
				case "document_id":
					if str := v.GetStringValue(); str != "" {
						result.DocumentID = str
					}
				default:
					result.Metadata[k] = extractValue(v)
				}
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// Close implements vectorstore.VectorStore.
func (c *Client) Close() error {
	return c.conn.Close()
}

// buildFilter converts SearchFilter to Qdrant Filter.
func buildFilter(filter vectorstore.SearchFilter) *qdrant.Filter {
	var conditions []*qdrant.Condition

	// Filter by source_id
	if filter.SourceID != "" {
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "source_id",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: filter.SourceID}},
				},
			},
		})
	}

	// Filter by metadata
	for key, value := range filter.Metadata {
		conditions = append(conditions, buildMatchCondition(key, value))
	}

	if len(conditions) == 0 {
		return nil
	}

	return &qdrant.Filter{Must: conditions}
}

// buildMatchCondition creates a match condition for a key-value pair.
func buildMatchCondition(key string, value any) *qdrant.Condition {
	var match *qdrant.Match

	switch v := value.(type) {
	case string:
		match = &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: v}}
	case int:
		match = &qdrant.Match{MatchValue: &qdrant.Match_Integer{Integer: int64(v)}}
	case int64:
		match = &qdrant.Match{MatchValue: &qdrant.Match_Integer{Integer: v}}
	case bool:
		match = &qdrant.Match{MatchValue: &qdrant.Match_Boolean{Boolean: v}}
	default:
		match = &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: fmt.Sprintf("%v", v)}}
	}

	return &qdrant.Condition{
		ConditionOneOf: &qdrant.Condition_Field{
			Field: &qdrant.FieldCondition{
				Key:   key,
				Match: match,
			},
		},
	}
}

// extractValue extracts a Go value from a Qdrant Value.
func extractValue(v *qdrant.Value) any {
	if v == nil {
		return nil
	}

	switch val := v.Kind.(type) {
	case *qdrant.Value_StringValue:
		return val.StringValue
	case *qdrant.Value_IntegerValue:
		return val.IntegerValue
	case *qdrant.Value_DoubleValue:
		return val.DoubleValue
	case *qdrant.Value_BoolValue:
		return val.BoolValue
	default:
		return nil
	}
}

// Compile-time check that Client implements VectorStore.
var _ vectorstore.VectorStore = (*Client)(nil)
