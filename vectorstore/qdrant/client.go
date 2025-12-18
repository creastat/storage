package qdrant

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/creastat/storage/vectorstore"
	"github.com/qdrant/go-client/qdrant"
)

// Config holds Qdrant connection configuration.
type Config struct {
	// URL is the Qdrant server address (e.g., "https://example.qdrant.io:6333").
	URL string

	// CollectionName is the name of the collection to search.
	CollectionName string

	// APIKey is optional API key for authentication.
	APIKey string
}

// Client implements vectorstore.VectorStore for Qdrant.
type Client struct {
	client         *qdrant.Client
	collectionName string
}

// New creates a new Qdrant client.
func New(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("qdrant url is required")
	}

	// Parse the URL to extract host, port, and scheme
	parsedURL := cfg.URL
	if !strings.HasPrefix(parsedURL, "http://") && !strings.HasPrefix(parsedURL, "https://") {
		parsedURL = "https://" + parsedURL
	}

	u, err := url.Parse(parsedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse qdrant url: %w", err)
	}

	// Extract host and port
	host := u.Hostname()
	port := 6334 // default port
	if u.Port() != "" {
		p, err := strconv.Atoi(u.Port())
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}
		port = p
	}

	useTLS := u.Scheme == "https"

	// Create Qdrant client
	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: cfg.APIKey,
		UseTLS: useTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant client: %w", err)
	}

	return &Client{
		client:         qdrantClient,
		collectionName: cfg.CollectionName,
	}, nil
}

// Search implements vectorstore.VectorStore.
func (c *Client) Search(ctx context.Context, vector []float32, filter vectorstore.SearchFilter, limit int) ([]vectorstore.SearchResult, error) {
	// Build Qdrant filter
	qdrantFilter := buildQdrantFilter(filter)

    // Perform search using Query method
    limitUint64 := uint64(limit)
    points, err := c.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: c.collectionName,
		Query:          qdrant.NewQuery(vector...),
        Limit:          &limitUint64,
		Filter:         qdrantFilter,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant search failed: %w", err)
	}

	// Convert results
	results := make([]vectorstore.SearchResult, 0, len(points))
	for _, point := range points {
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
	return c.client.Close()
}

// buildQdrantFilter converts SearchFilter to Qdrant Filter.
func buildQdrantFilter(filter vectorstore.SearchFilter) *qdrant.Filter {
	var conditions []*qdrant.Condition

	// Filter by source_id(s)
	if len(filter.SourceIDs) > 0 {
		if len(filter.SourceIDs) == 1 {
			// Single source ID
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key:   "source_id",
						Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: filter.SourceIDs[0]}},
					},
				},
			})
		} else {
			// Multiple source IDs
			keywords := make([]string, len(filter.SourceIDs))
			copy(keywords, filter.SourceIDs)
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "source_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keywords{
								Keywords: &qdrant.RepeatedStrings{Strings: keywords},
							},
						},
					},
				},
			})
		}
	} else if filter.SourceID != "" {
		// Backward compatibility: single source ID
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
