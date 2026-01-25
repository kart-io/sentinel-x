package biz

import (
	"context"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/rag/metrics"
	"github.com/kart-io/sentinel-x/internal/rag/store"
	"github.com/kart-io/sentinel-x/pkg/infra/pool"
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
	// 树检索组件（POC 阶段）
	treeRetriever *TreeRetriever // 树形检索器
	treeBuilder   *TreeBuilder   // 树构建器
	treeEnabled   bool           // 树功能开关
}

// ServiceConfig RAG 服务配置。
type ServiceConfig struct {
	IndexerConfig       *IndexerConfig
	RetrieverConfig     *RetrieverConfig
	GeneratorConfig     *GeneratorConfig
	QueryCacheConfig    *QueryCacheConfig
	TreeRetrieverConfig *TreeRetrieverConfig // 树检索配置
	TreeBuilderConfig   *TreeBuilderConfig   // 树构建配置
	TreeEnabled         bool                 // 树功能开关
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

	// 初始化树组件（如果启用）
	var treeRetriever *TreeRetriever
	var treeBuilder *TreeBuilder
	if config.TreeEnabled {
		if config.TreeRetrieverConfig != nil {
			treeRetriever = NewTreeRetriever(vectorStore, embedProvider, config.TreeRetrieverConfig)
		}
		if config.TreeBuilderConfig != nil {
			treeBuilder = NewTreeBuilder(vectorStore, embedProvider, chatProvider, config.TreeBuilderConfig)
		}
	}

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
		treeRetriever: treeRetriever,
		treeBuilder:   treeBuilder,
		treeEnabled:   config.TreeEnabled,
	}
}

// IndexFromURL 从 URL 下载并索引文档。
func (s *RAGService) IndexFromURL(ctx context.Context, url string) error {
	// 1. 索引文档
	if err := s.indexer.IndexFromURL(ctx, url); err != nil {
		return err
	}

	// 2. 异步构建树索引
	s.buildTreeAsync()

	return nil
}

// IndexDirectory 索引目录中的所有文档。
func (s *RAGService) IndexDirectory(ctx context.Context, dir string) error {
	// 1. 索引文档
	if err := s.indexer.IndexDirectory(ctx, dir); err != nil {
		return err
	}

	// 2. 异步构建树索引
	s.buildTreeAsync()

	return nil
}

// buildTreeAsync 异步构建树索引（后台任务）。
// 如果启用树功能，将在后台异步执行树构建，避免阻塞API响应。
func (s *RAGService) buildTreeAsync() {
	if !s.treeEnabled || s.treeBuilder == nil {
		return
	}

	logger.Info("树形索引将在后台异步构建")

	// 异步执行树构建，避免阻塞API响应
	treeTask := func() {
		// 延迟60秒，避免 Milvus Serverless 速率限制
		// 让索引阶段的写入配额恢复
		time.Sleep(60 * time.Second)

		logger.Info("开始构建树形索引（后台任务）...")
		// POC 阶段传空 documentID，表示为所有文档构建树
		if err := s.treeBuilder.BuildTree(context.Background(), ""); err != nil {
			logger.Warnw("树索引构建失败（后台）", "error", err.Error())
		} else {
			logger.Info("树形索引构建成功（后台）")
		}
	}

	// 提交到后台池，降级处理：池不可用时直接用 goroutine
	if err := pool.SubmitToType(pool.BackgroundPool, treeTask); err != nil {
		logger.Warnw("后台池不可用，降级到 goroutine",
			"error", err.Error(),
		)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorw("树构建任务 panic", "error", r)
				}
			}()
			treeTask()
		}()
	}
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
	var retrievalResult *RetrievalResult
	var err error

	// 树检索逻辑（带降级处理）
	if s.treeEnabled && s.treeRetriever != nil {
		// 使用树检索
		// 注意：POC 阶段传空 documentID，表示查询所有文档
		retrievalResult, err = s.treeRetriever.Retrieve(ctx, question, "")

		// 降级条件：错误 或 空结果
		if err != nil || (retrievalResult != nil && len(retrievalResult.Results) == 0) {
			if err != nil {
				logger.Warnw("树检索失败，降级到向量检索",
					"error", err.Error(),
					"question", question,
				)
			} else {
				logger.Warnw("树检索返回空结果，降级到向量检索",
					"question", question,
				)
			}
			// 降级到向量检索
			retrievalResult, err = s.retriever.Retrieve(ctx, question)
		}
	} else {
		// 使用向量检索（默认）
		retrievalResult, err = s.retriever.Retrieve(ctx, question)
	}

	retrievalDuration := time.Since(retrievalStart)
	s.metrics.RecordRetrieval(retrievalDuration, err)
	if err != nil {
		queryErr = err
		return nil, err
	}

	// 3. 生成答案
	llmStart := time.Now()
	resp, err := s.generator.GenerateAnswer(ctx, question, retrievalResult.Results)
	llmDuration := time.Since(llmStart)

	// 从响应中获取 token 使用信息
	promptTokens := 0
	completionTokens := 0
	if resp != nil && resp.TokenUsage != nil {
		promptTokens = resp.TokenUsage.PromptTokens
		completionTokens = resp.TokenUsage.CompletionTokens
	}
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
		Answer:  resp.Content,
		Sources: sources,
	}

	// 5. 写入缓存
	if s.cache != nil {
		// 缓存写入失败不影响正常返回,错误已在 cache.Set 中记录
		_ = s.cache.Set(ctx, question, queryResult)
	}

	// 记录缓存未命中的成功查询
	s.metrics.RecordQuery(false, nil)

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
