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
	// Level 树节点层级（0=叶子节点，1+=中间/根节点）。
	Level int
	// ParentID 父节点 ID（根节点为空字符串）。
	ParentID string
	// NodeType 节点类型（0=叶子，1=中间，2=根）。
	NodeType int
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
	// Metadata 检索结果的元数据（用于树形索引等扩展信息）。
	Metadata map[string]interface{}
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

	// SearchWithFilter 支持过滤表达式的向量搜索。
	// expr 为 Milvus 过滤表达式，例如：
	// - "level == 1" 查询第1层节点
	// - "parent_id == 'xxx'" 查询指定父节点的子节点
	// - "level == 0" 查询所有叶子节点
	SearchWithFilter(ctx context.Context, collection string, embedding []float32, expr string, topK int) ([]*SearchResult, error)

	// GetStats 获取集合统计信息。
	GetStats(ctx context.Context, collection string) (int64, error)

	// Close 关闭连接。
	Close(ctx context.Context) error
}
