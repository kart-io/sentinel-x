package store

import (
	"context"
)

// Chunk 表示文档块。
type Chunk struct {
	// ID 文档块 ID。
	ID string
	// DocumentID 所属文档 ID。
	DocumentID string
	// DocumentName 文档名称。
	DocumentName string
	// Section 所属章节。
	Section string
	// Content 文档内容。
	Content string
	// Embedding 嵌入向量。
	Embedding []float32
}

// SearchResult 表示检索结果。
type SearchResult struct {
	// ID 文档块 ID。
	ID string
	// DocumentID 所属文档 ID。
	DocumentID string
	// DocumentName 文档名称。
	DocumentName string
	// Section 所属章节。
	Section string
	// Content 文档内容。
	Content string
	// Score 相似度分数。
	Score float32
}

// CollectionConfig 集合配置。
type CollectionConfig struct {
	// Name 集合名称。
	Name string
	// Description 集合描述。
	Description string
	// Dimension 向量维度。
	Dimension int
}

// VectorStore 定义向量存储接口。
type VectorStore interface {
	// CreateCollection 创建集合。
	CreateCollection(ctx context.Context, config *CollectionConfig) error

	// Insert 批量插入文档块。
	Insert(ctx context.Context, collection string, chunks []*Chunk) ([]string, error)

	// Search 向量相似度搜索。
	Search(ctx context.Context, collection string, embedding []float32, topK int) ([]*SearchResult, error)

	// GetStats 获取集合统计信息。
	GetStats(ctx context.Context, collection string) (int64, error)

	// Close 关闭连接。
	Close(ctx context.Context) error
}
