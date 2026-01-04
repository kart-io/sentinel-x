package biz

import (
	"context"
	"time"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/rag/metrics"
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
	cache         *QueryCache
	store         store.VectorStore
	embedProvider llm.EmbeddingProvider
	chatProvider  llm.ChatProvider
	collection    string
	metrics       *metrics.RAGMetrics // 业务指标收集器
}

// ServiceConfig RAG 服务配置。
type ServiceConfig struct {
	IndexerConfig    *IndexerConfig
	RetrieverConfig  *RetrieverConfig
	GeneratorConfig  *GeneratorConfig
	QueryCacheConfig *QueryCacheConfig
}

// NewRAGService 创建 RAG 服务实例。
func NewRAGService(
	vectorStore store.VectorStore,
	embedProvider llm.EmbeddingProvider,
	chatProvider llm.ChatProvider,
	cache *QueryCache,
	config *ServiceConfig,
) *RAGService {
	indexer := NewIndexer(vectorStore, embedProvider, config.IndexerConfig)
	retriever := NewRetriever(vectorStore, embedProvider, chatProvider, config.RetrieverConfig)
	generator := NewGenerator(chatProvider, config.GeneratorConfig)

	return &RAGService{
		indexer:       indexer,
		retriever:     retriever,
		generator:     generator,
		cache:         cache,
		store:         vectorStore,
		embedProvider: embedProvider,
		chatProvider:  chatProvider,
		collection:    config.IndexerConfig.Collection,
		metrics:       metrics.GetRAGMetrics(), // 初始化全局指标实例
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
	var queryErr error
	defer func() {
		// 记录查询指标（缓存命中/未命中在下面分别记录）
		if queryErr != nil {
			s.metrics.RecordQuery(false, queryErr)
		}
	}()

	// 1. 尝试从缓存获取
	cacheHit := false
	if s.cache != nil {
		cachedResult, err := s.cache.Get(ctx, question)
		if err == nil && cachedResult != nil {
			// 缓存命中，直接返回
			s.metrics.RecordQuery(true, nil)
			return cachedResult, nil
		}
		// 缓存未命中或出错，继续正常流程
	}

	// 2. 检索相关文档
	retrievalStart := time.Now()
	retrievalResult, err := s.retriever.Retrieve(ctx, question)
	retrievalDuration := time.Since(retrievalStart)
	s.metrics.RecordRetrieval(retrievalDuration, err)
	if err != nil {
		queryErr = err
		return nil, err
	}

	// 3. 生成答案
	llmStart := time.Now()
	answer, err := s.generator.GenerateAnswer(ctx, question, retrievalResult.Results)
	llmDuration := time.Since(llmStart)

	// TODO: 从 generator 获取实际 token 数量（需要修改 Generator 接口返回 token 信息）
	// 暂时使用估算值
	promptTokens := 0     // 需要从 generator 传递
	completionTokens := 0 // 需要从 generator 传递
	s.metrics.RecordLLMCall(llmDuration, promptTokens, completionTokens, err)

	if err != nil {
		queryErr = err
		return nil, err
	}

	// 4. 构建响应
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

	queryResult := &model.QueryResult{
		Answer:  answer,
		Sources: sources,
	}

	// 5. 写入缓存
	if s.cache != nil {
		// 缓存写入失败不影响正常返回,错误已在 cache.Set 中记录
		_ = s.cache.Set(ctx, question, queryResult)
	}

	// 记录缓存未命中的成功查询
	s.metrics.RecordQuery(cacheHit, nil)

	return queryResult, nil
}

// GetStats 获取知识库统计信息。
func (s *RAGService) GetStats(ctx context.Context) (map[string]any, error) {
	count, err := s.store.GetStats(ctx, s.collection)
	if err != nil {
		return nil, err
	}

	stats := map[string]any{
		"collection":     s.collection,
		"chunk_count":    count,
		"embed_provider": s.embedProvider.Name(),
		"chat_provider":  s.chatProvider.Name(),
	}

	// 添加缓存统计信息
	if s.cache != nil {
		cacheStats, err := s.cache.GetStats(ctx)
		if err == nil {
			stats["cache"] = cacheStats
		}
	}

	// 添加业务指标统计
	stats["metrics"] = s.metrics.Stats()

	return stats, nil
}

// 确保 RAGService 实现了 Service 接口。
var _ Service = (*RAGService)(nil)
