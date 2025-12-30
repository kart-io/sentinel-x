package milvus

import (
	"context"
	"fmt"
	"strconv"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"

	milvusopts "github.com/kart-io/sentinel-x/pkg/options/milvus"
)

// Client wraps the Milvus SDK client.
type Client struct {
	client *milvusclient.Client
	opts   *milvusopts.Options
}

// New creates a new Milvus client.
func New(opts *milvusopts.Options) (*Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("milvus options is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	c, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address:  opts.Address,
		Username: opts.Username,
		Password: opts.Password,
		DBName:   opts.Database,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to milvus: %w", err)
	}

	return &Client{
		client: c,
		opts:   opts,
	}, nil
}

// Close closes the Milvus client connection.
func (c *Client) Close(ctx context.Context) error {
	return c.client.Close(ctx)
}

// RawClient returns the underlying Milvus client.
func (c *Client) RawClient() *milvusclient.Client {
	return c.client
}

// CollectionSchema defines the schema for a vector collection.
type CollectionSchema struct {
	Name        string
	Description string
	Dimension   int
	MetaFields  []MetaField
}

// MetaField defines a metadata field in the collection.
type MetaField struct {
	Name     string
	DataType entity.FieldType
	MaxLen   int // For VARCHAR type
}

// CreateCollection creates a new collection with the given schema.
func (c *Client) CreateCollection(ctx context.Context, schema *CollectionSchema) error {
	// Check if collection exists
	exists, err := c.client.HasCollection(ctx, milvusclient.NewHasCollectionOption(schema.Name))
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}
	if exists {
		return nil // Collection already exists
	}

	// Build schema using the new SDK
	collSchema := entity.NewSchema().
		WithName(schema.Name).
		WithDescription(schema.Description).
		WithAutoID(true)

	// Add primary key field
	collSchema.WithField(
		entity.NewField().
			WithName("id").
			WithDataType(entity.FieldTypeInt64).
			WithIsPrimaryKey(true).
			WithIsAutoID(true),
	)

	// Add vector field
	collSchema.WithField(
		entity.NewField().
			WithName("embedding").
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(int64(schema.Dimension)),
	)

	// Add metadata fields
	for _, f := range schema.MetaFields {
		field := entity.NewField().
			WithName(f.Name).
			WithDataType(f.DataType)
		if f.DataType == entity.FieldTypeVarChar && f.MaxLen > 0 {
			field.WithMaxLength(int64(f.MaxLen))
		}
		collSchema.WithField(field)
	}

	// Create collection
	if err := c.client.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(schema.Name, collSchema)); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Create index for vector field - using IVF_FLAT
	idx := index.NewIvfFlatIndex(entity.L2, 128)
	createIdxTask, err := c.client.CreateIndex(ctx, milvusclient.NewCreateIndexOption(schema.Name, "embedding", idx))
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	if err := createIdxTask.Await(ctx); err != nil {
		return fmt.Errorf("failed to wait for index creation: %w", err)
	}

	// Load collection into memory
	loadTask, err := c.client.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(schema.Name))
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}
	if err := loadTask.Await(ctx); err != nil {
		return fmt.Errorf("failed to wait for collection loading: %w", err)
	}

	return nil
}

// InsertData represents data to be inserted into a collection.
type InsertData struct {
	Embeddings [][]float32
	Metadata   map[string][]any
}

// Insert inserts vectors and metadata into the collection.
func (c *Client) Insert(ctx context.Context, collectionName string, data *InsertData) ([]int64, error) {
	// Build columns for insertion
	columns := make([]column.Column, 0, len(data.Metadata)+1)

	// Add vector column
	columns = append(columns, column.NewColumnFloatVector("embedding", len(data.Embeddings[0]), data.Embeddings))

	// Add metadata columns
	for name, values := range data.Metadata {
		switch v := values[0].(type) {
		case string:
			strVals := make([]string, len(values))
			for i, val := range values {
				strVals[i] = val.(string)
			}
			columns = append(columns, column.NewColumnVarChar(name, strVals))
		case int64:
			intVals := make([]int64, len(values))
			for i, val := range values {
				intVals[i] = val.(int64)
			}
			columns = append(columns, column.NewColumnInt64(name, intVals))
		default:
			return nil, fmt.Errorf("unsupported metadata type: %T for field %s", v, name)
		}
	}

	// Insert using new SDK API
	result, err := c.client.Insert(ctx, milvusclient.NewColumnBasedInsertOption(collectionName, columns...))
	if err != nil {
		return nil, fmt.Errorf("failed to insert data: %w", err)
	}

	// Flush to ensure data is visible immediately
	// Note: In production, frequent flushing might impact performance, but it's useful for RAG ingestion tasks
	flushTask, err := c.client.Flush(ctx, milvusclient.NewFlushOption(collectionName))
	if err != nil {
		return nil, fmt.Errorf("failed to flush collection: %w", err)
	}
	if err := flushTask.Await(ctx); err != nil {
		return nil, fmt.Errorf("failed to wait for flush: %w", err)
	}

	// Extract IDs from result
	ids := result.IDs.(*column.ColumnInt64).Data()
	return ids, nil
}

// SearchResult represents a single search result.
type SearchResult struct {
	ID       int64
	Score    float32
	Metadata map[string]any
}

// Search performs a vector similarity search.
func (c *Client) Search(ctx context.Context, collectionName string, vector []float32, topK int, outputFields []string) ([]SearchResult, error) {
	// Ensure collection is loaded
	loadTask, err := c.client.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(collectionName))
	if err != nil {
		return nil, fmt.Errorf("failed to load collection: %w", err)
	}
	if err := loadTask.Await(ctx); err != nil {
		return nil, fmt.Errorf("failed to wait for collection loading: %w", err)
	}

	// Build search request
	searchVectors := []entity.Vector{entity.FloatVector(vector)}

	results, err := c.client.Search(ctx, milvusclient.NewSearchOption(
		collectionName,
		topK,
		searchVectors,
	).WithANNSField("embedding").
		WithSearchParam("nprobe", "16").
		WithOutputFields(outputFields...))
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	if len(results) == 0 {
		return []SearchResult{}, nil
	}

	// Parse results
	searchResults := make([]SearchResult, 0, results[0].ResultCount)
	for i := 0; i < results[0].ResultCount; i++ {
		result := SearchResult{
			Score:    results[0].Scores[i],
			Metadata: make(map[string]any),
		}

		// Try to extract ID
		if idCol, ok := results[0].IDs.(*column.ColumnInt64); ok {
			result.ID = idCol.Data()[i]
		}

		// Extract metadata from output fields
		for _, field := range results[0].Fields {
			switch col := field.(type) {
			case *column.ColumnVarChar:
				// Use Value(i) to safely get value, or Data()[i]
				result.Metadata[col.Name()] = col.Data()[i]
			case *column.ColumnInt64:
				result.Metadata[col.Name()] = col.Data()[i]
			}
		}

		searchResults = append(searchResults, result)
	}

	return searchResults, nil
}

// DeleteByIDs deletes vectors by their IDs.
func (c *Client) DeleteByIDs(ctx context.Context, collectionName string, ids []int64) error {
	idColumn := column.NewColumnInt64("id", ids)
	// Delete returns (DeleteResult, error) in v2
	if _, err := c.client.Delete(ctx, milvusclient.NewDeleteOption(collectionName).WithInt64IDs("id", idColumn.Data())); err != nil {
		return fmt.Errorf("failed to delete by ids: %w", err)
	}
	return nil
}

// DropCollection drops a collection.
func (c *Client) DropCollection(ctx context.Context, collectionName string) error {
	if err := c.client.DropCollection(ctx, milvusclient.NewDropCollectionOption(collectionName)); err != nil {
		return fmt.Errorf("failed to drop collection: %w", err)
	}
	return nil
}

// GetCollectionStats returns the number of entities in a collection.
func (c *Client) GetCollectionStats(ctx context.Context, collectionName string) (int64, error) {
	stats, err := c.client.GetCollectionStats(ctx, milvusclient.NewGetCollectionStatsOption(collectionName))
	if err != nil {
		return 0, fmt.Errorf("failed to get collection stats: %w", err)
	}

	if val, ok := stats["row_count"]; ok {
		return strconv.ParseInt(val, 10, 64)
	}
	return 0, nil
}
