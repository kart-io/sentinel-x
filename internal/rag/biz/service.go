package biz

import (
	"context"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/rag/store"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// Service 定义 RAG 服务接口。
type Service interface {
	// IndexFromURL 从 URL 下载并索引文档。
	IndexFromURL(ctx context.Context, url string) error
	// IndexDirectory 索引目录中的所有文档。
	IndexDirectory(ctx context.Context, dir string) error
	// Query 执行 RAG 查询。
	Query(ctx context.Context, question string) (*model.QueryResult, error)
	// GetStats 获取知识库统计信息。
	GetStats(ctx context.Context) (map[string]any, error)
}

// RAGService 组合 Indexer、Retriever 和 Generator 提供完整的 RAG 服务。
type RAGService struct {
	indexer       *Indexer
	retriever     *Retriever
	generator     *Generator
	store         store.VectorStore
	embedProvider llm.EmbeddingProvider
	chatProvider  llm.ChatProvider
	collection    string
}

// ServiceConfig RAG 服务配置。
type ServiceConfig struct {
	IndexerConfig   *IndexerConfig
	RetrieverConfig *RetrieverConfig
	GeneratorConfig *GeneratorConfig
}

// NewRAGService 创建 RAG 服务实例。
func NewRAGService(
	vectorStore store.VectorStore,
	embedProvider llm.EmbeddingProvider,
	chatProvider llm.ChatProvider,
	config *ServiceConfig,
) *RAGService {
	indexer := NewIndexer(vectorStore, embedProvider, config.IndexerConfig)
	retriever := NewRetriever(vectorStore, embedProvider, chatProvider, config.RetrieverConfig)
	generator := NewGenerator(chatProvider, config.GeneratorConfig)

	return &RAGService{
		indexer:       indexer,
		retriever:     retriever,
		generator:     generator,
		store:         vectorStore,
		embedProvider: embedProvider,
		chatProvider:  chatProvider,
		collection:    config.IndexerConfig.Collection,
	}
}

// IndexFromURL 从 URL 下载并索引文档。
func (s *RAGService) IndexFromURL(ctx context.Context, url string) error {
	return s.indexer.IndexFromURL(ctx, url)
}

// IndexDirectory 索引目录中的所有文档。
func (s *RAGService) IndexDirectory(ctx context.Context, dir string) error {
	return s.indexer.IndexDirectory(ctx, dir)
}

// Query 执行 RAG 查询。
func (s *RAGService) Query(ctx context.Context, question string) (*model.QueryResult, error) {
	// 1. 检索相关文档
	retrievalResult, err := s.retriever.Retrieve(ctx, question)
	if err != nil {
		return nil, err
	}

	// 2. 生成答案
	answer, err := s.generator.GenerateAnswer(ctx, question, retrievalResult.Results)
	if err != nil {
		return nil, err
	}

	// 3. 构建响应
	sources := make([]model.ChunkSource, len(retrievalResult.Results))
	for i, result := range retrievalResult.Results {
		sources[i] = model.ChunkSource{
			DocumentID:   result.DocumentID,
			DocumentName: result.DocumentName,
			Section:      result.Section,
			Content:      result.Content,
			Score:        result.Score,
		}
	}

	return &model.QueryResult{
		Answer:  answer,
		Sources: sources,
	}, nil
}

// GetStats 获取知识库统计信息。
func (s *RAGService) GetStats(ctx context.Context) (map[string]any, error) {
	count, err := s.store.GetStats(ctx, s.collection)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"collection":     s.collection,
		"chunk_count":    count,
		"embed_provider": s.embedProvider.Name(),
		"chat_provider":  s.chatProvider.Name(),
	}, nil
}

// 确保 RAGService 实现了 Service 接口。
var _ Service = (*RAGService)(nil)
