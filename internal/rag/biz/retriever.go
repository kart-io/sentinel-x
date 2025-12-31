package biz

import (
	"context"
	"fmt"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/enhancer"
	"github.com/kart-io/sentinel-x/internal/rag/store"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// RetrieverConfig 检索器配置。
type RetrieverConfig struct {
	// TopK 返回的结果数量。
	TopK int
	// Collection 集合名称。
	Collection string
	// Enhancer 增强器配置。
	Enhancer enhancer.Config
}

// RetrievalResult 表示检索结果。
type RetrievalResult struct {
	// Query 增强后的查询。
	Query string
	// Results 检索结果列表。
	Results []*store.SearchResult
}

// Retriever 负责文档检索。
type Retriever struct {
	store         store.VectorStore
	embedProvider llm.EmbeddingProvider
	enhancer      *enhancer.Enhancer
	config        *RetrieverConfig
}

// NewRetriever 创建检索器实例。
func NewRetriever(
	vectorStore store.VectorStore,
	embedProvider llm.EmbeddingProvider,
	chatProvider llm.ChatProvider,
	config *RetrieverConfig,
) *Retriever {
	ragEnhancer := enhancer.New(chatProvider, embedProvider, config.Enhancer)
	return &Retriever{
		store:         vectorStore,
		embedProvider: embedProvider,
		enhancer:      ragEnhancer,
		config:        config,
	}
}

// Retrieve 执行检索。
func (r *Retriever) Retrieve(ctx context.Context, question string) (*RetrievalResult, error) {
	logger.Infof("Processing query: %s", question)

	// 1. 增强查询（查询重写 + HyDE）
	enhancedQuery, embeddings, err := r.enhancer.EnhanceQuery(ctx, question)
	if err != nil {
		logger.Warnw("查询增强失败，使用原始查询", "error", err.Error())
		questionEmbed, embedErr := r.embedProvider.EmbedSingle(ctx, question)
		if embedErr != nil {
			return nil, fmt.Errorf("failed to embed question: %w", embedErr)
		}
		embeddings = [][]float32{questionEmbed}
		enhancedQuery = question
	}

	// 2. 执行检索（支持多嵌入检索）
	var allResults []enhancer.SearchResult
	for _, embedding := range embeddings {
		results, err := r.store.Search(ctx, r.config.Collection, embedding, r.config.TopK)
		if err != nil {
			logger.Warnw("检索失败", "error", err.Error())
			continue
		}

		for _, res := range results {
			allResults = append(allResults, enhancer.SearchResult{
				ID:      res.ID,
				Content: res.Content,
				Score:   res.Score,
				Metadata: map[string]any{
					"document_id":   res.DocumentID,
					"document_name": res.DocumentName,
					"section":       res.Section,
				},
			})
		}
	}

	// 合并多次检索结果（如果启用了 HyDE）
	if len(embeddings) > 1 {
		allResults = enhancer.MergeEmbeddingResults([][]enhancer.SearchResult{allResults})
	}

	if len(allResults) == 0 {
		return &RetrievalResult{
			Query:   enhancedQuery,
			Results: []*store.SearchResult{},
		}, nil
	}

	// 3. 重排序检索结果
	rerankedResults, err := r.enhancer.RerankResults(ctx, enhancedQuery, allResults)
	if err != nil {
		logger.Warnw("重排序失败，使用原始结果", "error", err.Error())
		rerankedResults = allResults
	}

	// 4. 文档重组（高置信度放首尾）
	repackedResults := r.enhancer.RepackDocuments(rerankedResults)

	// 转换为 store.SearchResult
	storeResults := make([]*store.SearchResult, len(repackedResults))
	for i, res := range repackedResults {
		storeResults[i] = &store.SearchResult{
			ID:           res.ID,
			DocumentID:   res.Metadata["document_id"].(string),
			DocumentName: res.Metadata["document_name"].(string),
			Section:      res.Metadata["section"].(string),
			Content:      res.Content,
			Score:        res.Score,
		}
	}

	return &RetrievalResult{
		Query:   enhancedQuery,
		Results: storeResults,
	}, nil
}
