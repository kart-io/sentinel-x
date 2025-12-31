package store

import (
	"context"
	"fmt"

	"github.com/kart-io/sentinel-x/pkg/component/milvus"
	"github.com/milvus-io/milvus/client/v2/entity"
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
	}

	for i, chunk := range chunks {
		embeddings[i] = chunk.Embedding
		metadata["document_id"][i] = chunk.DocumentID
		metadata["document_name"][i] = chunk.DocumentName
		metadata["section"][i] = chunk.Section
		metadata["content"][i] = chunk.Content
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
	outputFields := []string{"document_id", "document_name", "section", "content"}
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
