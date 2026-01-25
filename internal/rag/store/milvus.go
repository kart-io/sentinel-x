package store

import (
	"context"
	"fmt"

	"github.com/kart-io/sentinel-x/pkg/component/milvus"
	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// MilvusStore 实现基于 Milvus 的向量存储。
type MilvusStore struct {
	client *milvus.Client
}

// NewMilvusStore 创建 Milvus 存储实例。
func NewMilvusStore(client *milvus.Client) *MilvusStore {
	return &MilvusStore{client: client}
}

// CreateCollection 创建 Milvus 集合。
func (s *MilvusStore) CreateCollection(ctx context.Context, config *CollectionConfig) error {
	schema := &milvus.CollectionSchema{
		Name:        config.Name,
		Description: config.Description,
		Dimension:   config.Dimension,
		MetaFields: []milvus.MetaField{
			{Name: "document_id", DataType: entity.FieldTypeVarChar, MaxLen: 64},
			{Name: "document_name", DataType: entity.FieldTypeVarChar, MaxLen: 255},
			{Name: "section", DataType: entity.FieldTypeVarChar, MaxLen: 255},
			{Name: "content", DataType: entity.FieldTypeVarChar, MaxLen: 65535},
			// 树形索引字段
			{Name: "level", DataType: entity.FieldTypeInt64},
			{Name: "parent_id", DataType: entity.FieldTypeVarChar, MaxLen: 64},
			{Name: "node_type", DataType: entity.FieldTypeInt64},
		},
	}
	return s.client.CreateCollection(ctx, schema)
}

// Insert 批量插入文档块到 Milvus。
func (s *MilvusStore) Insert(ctx context.Context, collection string, chunks []*Chunk) ([]string, error) {
	if len(chunks) == 0 {
		return nil, nil
	}

	embeddings := make([][]float32, len(chunks))
	metadata := map[string][]any{
		"document_id":   make([]any, len(chunks)),
		"document_name": make([]any, len(chunks)),
		"section":       make([]any, len(chunks)),
		"content":       make([]any, len(chunks)),
		// 树形索引字段
		"level":     make([]any, len(chunks)),
		"parent_id": make([]any, len(chunks)),
		"node_type": make([]any, len(chunks)),
	}

	for i, chunk := range chunks {
		embeddings[i] = chunk.Embedding
		metadata["document_id"][i] = chunk.DocumentID
		metadata["document_name"][i] = chunk.DocumentName
		metadata["section"][i] = chunk.Section
		metadata["content"][i] = chunk.Content
		// 树形索引字段（int 转 int64）
		metadata["level"][i] = int64(chunk.Level)
		metadata["parent_id"][i] = chunk.ParentID
		metadata["node_type"][i] = int64(chunk.NodeType)
	}

	data := &milvus.InsertData{
		Embeddings: embeddings,
		Metadata:   metadata,
	}

	ids, err := s.client.Insert(ctx, collection, data)
	if err != nil {
		return nil, fmt.Errorf("failed to insert into milvus: %w", err)
	}

	// 将 int64 ID 转换为 string
	stringIDs := make([]string, len(ids))
	for i, id := range ids {
		stringIDs[i] = fmt.Sprintf("%d", id)
	}

	return stringIDs, nil
}

// Search 执行向量相似度搜索。
func (s *MilvusStore) Search(ctx context.Context, collection string, embedding []float32, topK int) ([]*SearchResult, error) {
	outputFields := []string{"document_id", "document_name", "section", "content", "level", "parent_id", "node_type"}
	results, err := s.client.Search(ctx, collection, embedding, topK, outputFields)
	if err != nil {
		return nil, fmt.Errorf("failed to search milvus: %w", err)
	}

	searchResults := make([]*SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = &SearchResult{
			ID:           r.Metadata["document_id"].(string),
			DocumentID:   r.Metadata["document_id"].(string),
			DocumentName: r.Metadata["document_name"].(string),
			Section:      r.Metadata["section"].(string),
			Content:      r.Metadata["content"].(string),
			Score:        r.Score,
		}
	}

	return searchResults, nil
}

// SearchWithFilter 执行支持过滤表达式的向量相似度搜索。
// expr 为 Milvus 过滤表达式，例如 "level == 1" 或 "parent_id == 'xxx'"。
func (s *MilvusStore) SearchWithFilter(ctx context.Context, collection string, embedding []float32, expr string, topK int) ([]*SearchResult, error) {
	outputFields := []string{"document_id", "document_name", "section", "content", "level", "parent_id", "node_type"}

	// 使用底层 Milvus 客户端的 Search 方法，并添加过滤表达式
	// 注意：这里需要直接使用 milvus client 的 Search API 并传入 filter
	rawClient := s.client.RawClient()
	if rawClient == nil {
		return nil, fmt.Errorf("milvus client not initialized")
	}

	// 确保集合已加载
	loadTask, err := rawClient.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(collection))
	if err != nil {
		return nil, fmt.Errorf("failed to load collection: %w", err)
	}
	if err := loadTask.Await(ctx); err != nil {
		return nil, fmt.Errorf("failed to wait for collection loading: %w", err)
	}

	// 构建搜索向量
	searchVectors := []entity.Vector{entity.FloatVector(embedding)}

	// 执行搜索（带过滤表达式）
	results, err := rawClient.Search(ctx, milvusclient.NewSearchOption(
		collection,
		topK,
		searchVectors,
	).WithANNSField("embedding").
		WithSearchParam("ef", "64").
		WithFilter(expr).
		WithOutputFields(outputFields...))
	if err != nil {
		return nil, fmt.Errorf("failed to search with filter: %w", err)
	}

	if len(results) == 0 {
		return []*SearchResult{}, nil
	}

	// 解析结果
	searchResults := make([]*SearchResult, 0, results[0].ResultCount)
	for i := 0; i < results[0].ResultCount; i++ {
		result := &SearchResult{
			Score: results[0].Scores[i],
		}

		// 提取字段值
		for _, field := range results[0].Fields {
			if col, ok := field.(*column.ColumnVarChar); ok {
				switch col.Name() {
				case "document_id":
					result.ID = col.Data()[i]
					result.DocumentID = col.Data()[i]
				case "document_name":
					result.DocumentName = col.Data()[i]
				case "section":
					result.Section = col.Data()[i]
				case "content":
					result.Content = col.Data()[i]
				}
			}
		}

		searchResults = append(searchResults, result)
	}

	return searchResults, nil
}

// GetStats 获取集合统计信息。
func (s *MilvusStore) GetStats(ctx context.Context, collection string) (int64, error) {
	return s.client.GetCollectionStats(ctx, collection)
}

// Close 关闭 Milvus 连接。
func (s *MilvusStore) Close(ctx context.Context) error {
	return s.client.Close(ctx)
}

// 确保 MilvusStore 实现了 VectorStore 接口。
var _ VectorStore = (*MilvusStore)(nil)
