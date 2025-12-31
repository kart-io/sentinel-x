// Package enhancer 提供 RAG 检索增强功能。
//
// 基于 Milvus 官方最佳实践和学术研究，该包实现了以下增强技术：
//   - Query Rewriting（查询重写）: 优化原始查询以提高检索精度
//   - HyDE（假设文档嵌入）: 生成假设文档来增强检索
//   - Reranking（重排序）: 对检索结果进行精细排序
//   - Document Repacking（文档重组）: 优化上下文顺序以提高 LLM 推理效果
//
// 参考文档：
//   - https://milvus.io/docs/how_to_enhance_your_rag.md
//   - https://arxiv.org/pdf/2407.01219 (Best Practices in RAG)
package enhancer

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/textutil"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// Config 增强器配置。
type Config struct {
	// EnableQueryRewrite 是否启用查询重写。
	EnableQueryRewrite bool `mapstructure:"enable_query_rewrite"`

	// EnableHyDE 是否启用 HyDE（假设文档嵌入）。
	EnableHyDE bool `mapstructure:"enable_hyde"`

	// EnableRerank 是否启用重排序。
	EnableRerank bool `mapstructure:"enable_rerank"`

	// EnableRepacking 是否启用文档重组。
	EnableRepacking bool `mapstructure:"enable_repacking"`

	// RerankTopK 重排序后保留的文档数量。
	RerankTopK int `mapstructure:"rerank_top_k"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() Config {
	return Config{
		EnableQueryRewrite: true,
		EnableHyDE:         false, // HyDE 增加延迟，默认关闭
		EnableRerank:       true,
		EnableRepacking:    true,
		RerankTopK:         5,
	}
}

// SearchResult 表示检索结果。
type SearchResult struct {
	// ID 文档 ID。
	ID string
	// Content 文档内容。
	Content string
	// Score 相似度分数。
	Score float32
	// Metadata 元数据。
	Metadata map[string]any
}

// Enhancer 提供 RAG 增强功能。
type Enhancer struct {
	chatProvider  llm.ChatProvider
	embedProvider llm.EmbeddingProvider
	config        Config
}

// New 创建新的增强器。
func New(chatProvider llm.ChatProvider, embedProvider llm.EmbeddingProvider, config Config) *Enhancer {
	return &Enhancer{
		chatProvider:  chatProvider,
		embedProvider: embedProvider,
		config:        config,
	}
}

// EnhanceQuery 增强查询，返回优化后的查询或假设文档嵌入。
// 返回值：(重写后的查询, 用于检索的嵌入向量列表, 错误)
func (e *Enhancer) EnhanceQuery(ctx context.Context, query string) (string, [][]float32, error) {
	enhancedQuery := query
	var embeddings [][]float32

	// 1. 查询重写
	if e.config.EnableQueryRewrite {
		rewritten, err := e.rewriteQuery(ctx, query)
		if err != nil {
			logger.Warnw("查询重写失败，使用原始查询", "error", err.Error())
		} else {
			enhancedQuery = rewritten
			logger.Debugw("查询已重写", "original", query, "rewritten", rewritten)
		}
	}

	// 2. 生成查询嵌入
	queryEmbed, err := e.embedProvider.EmbedSingle(ctx, enhancedQuery)
	if err != nil {
		return enhancedQuery, nil, fmt.Errorf("生成查询嵌入失败: %w", err)
	}
	embeddings = append(embeddings, queryEmbed)

	// 3. HyDE：生成假设文档并获取其嵌入
	if e.config.EnableHyDE {
		hydeEmbed, err := e.generateHyDEEmbedding(ctx, query)
		if err != nil {
			logger.Warnw("HyDE 生成失败", "error", err.Error())
		} else {
			embeddings = append(embeddings, hydeEmbed)
			logger.Debug("HyDE 嵌入已生成")
		}
	}

	return enhancedQuery, embeddings, nil
}

// RerankResults 对检索结果进行重排序。
func (e *Enhancer) RerankResults(ctx context.Context, query string, results []SearchResult) ([]SearchResult, error) {
	if !e.config.EnableRerank || len(results) == 0 {
		return results, nil
	}

	// 使用 LLM 评估每个结果与查询的相关性
	rerankedResults := make([]SearchResult, len(results))
	copy(rerankedResults, results)

	for i := range rerankedResults {
		score, err := e.scoreRelevance(ctx, query, rerankedResults[i].Content)
		if err != nil {
			logger.Warnw("相关性评分失败", "error", err.Error())
			continue
		}
		// 结合原始分数和 LLM 评分
		rerankedResults[i].Score = float32(0.3)*rerankedResults[i].Score + float32(0.7)*float32(score)
	}

	// 按新分数排序
	sort.Slice(rerankedResults, func(i, j int) bool {
		return rerankedResults[i].Score > rerankedResults[j].Score
	})

	// 截取 TopK
	if e.config.RerankTopK > 0 && len(rerankedResults) > e.config.RerankTopK {
		rerankedResults = rerankedResults[:e.config.RerankTopK]
	}

	logger.Debugw("重排序完成", "original_count", len(results), "final_count", len(rerankedResults))
	return rerankedResults, nil
}

// RepackDocuments 重组文档顺序，将高置信度文档放在首尾。
// 基于 "Lost in the Middle" 研究：LLM 更关注首尾内容。
func (e *Enhancer) RepackDocuments(results []SearchResult) []SearchResult {
	if !e.config.EnableRepacking || len(results) <= 2 {
		return results
	}

	// 按分数排序
	sorted := make([]SearchResult, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	// 重组：高分在首尾，低分在中间
	repacked := make([]SearchResult, len(sorted))
	left, right := 0, len(sorted)-1

	for i, doc := range sorted {
		if i%2 == 0 {
			repacked[left] = doc
			left++
		} else {
			repacked[right] = doc
			right--
		}
	}

	logger.Debug("文档已重组，高置信度内容放置在首尾")
	return repacked
}

// rewriteQuery 使用 LLM 重写查询以提高检索效果。
func (e *Enhancer) rewriteQuery(ctx context.Context, query string) (string, error) {
	prompt := fmt.Sprintf(`你是一个查询优化专家。请将以下用户查询重写为更适合向量检索的形式。

要求：
1. 保持原始查询的核心意图
2. 扩展关键术语和同义词
3. 使查询更加具体和明确
4. 添加相关上下文信息
5. 输出重写后的查询，不要包含任何解释

原始查询：%s

重写后的查询：`, query)

	response, err := e.chatProvider.Generate(ctx, prompt, "")
	if err != nil {
		return query, err
	}

	rewritten := strings.TrimSpace(response)
	if rewritten == "" {
		return query, nil
	}

	return rewritten, nil
}

// generateHyDEEmbedding 生成假设文档嵌入。
// HyDE (Hypothetical Document Embeddings) 通过 LLM 生成假设答案，
// 然后用假设答案的嵌入来检索相关文档。
func (e *Enhancer) generateHyDEEmbedding(ctx context.Context, query string) ([]float32, error) {
	prompt := fmt.Sprintf(`请针对以下问题，生成一段假设性的答案文档。
这段文档应该包含回答该问题所需的关键信息和技术细节。

问题：%s

假设文档：`, query)

	response, err := e.chatProvider.Generate(ctx, prompt, "")
	if err != nil {
		return nil, fmt.Errorf("生成假设文档失败: %w", err)
	}

	hypotheticalDoc := strings.TrimSpace(response)
	if hypotheticalDoc == "" {
		return nil, fmt.Errorf("生成的假设文档为空")
	}

	// 生成假设文档的嵌入
	embedding, err := e.embedProvider.EmbedSingle(ctx, hypotheticalDoc)
	if err != nil {
		return nil, fmt.Errorf("生成假设文档嵌入失败: %w", err)
	}

	return embedding, nil
}

// scoreRelevance 使用 LLM 评估文档与查询的相关性。
func (e *Enhancer) scoreRelevance(ctx context.Context, query, document string) (float64, error) {
	// 截断过长的文档
	truncatedDoc := textutil.TruncateString(document, 2000)

	prompt := fmt.Sprintf(`评估以下文档与查询的相关性。

查询：%s

文档：%s

请只返回一个 0 到 1 之间的数字，表示相关性分数：
- 1.0：完全相关，直接回答了查询
- 0.7-0.9：高度相关，包含大部分所需信息
- 0.4-0.6：部分相关，包含一些相关信息
- 0.1-0.3：低相关，只有少量相关内容
- 0.0：完全不相关

相关性分数：`, query, truncatedDoc)

	response, err := e.chatProvider.Generate(ctx, prompt, "")
	if err != nil {
		return 0.5, err
	}

	// 解析分数
	score := parseScore(response)
	return score, nil
}

// parseScore 从 LLM 响应中解析分数。
func parseScore(response string) float64 {
	response = strings.TrimSpace(response)

	// 尝试直接解析数字
	var score float64
	if _, err := fmt.Sscanf(response, "%f", &score); err == nil {
		if score >= 0 && score <= 1 {
			return score
		}
	}

	// 尝试提取数字
	for _, part := range strings.Fields(response) {
		if _, err := fmt.Sscanf(part, "%f", &score); err == nil {
			if score >= 0 && score <= 1 {
				return score
			}
		}
	}

	// 默认返回中等分数
	return 0.5
}

// MergeEmbeddingResults 合并多个嵌入检索的结果。
// 用于合并原始查询和 HyDE 查询的检索结果。
func MergeEmbeddingResults(resultSets [][]SearchResult) []SearchResult {
	if len(resultSets) == 0 {
		return nil
	}
	if len(resultSets) == 1 {
		return resultSets[0]
	}

	// 使用 RRF (Reciprocal Rank Fusion) 合并结果
	scoreMap := make(map[string]float64)
	resultMap := make(map[string]SearchResult)
	k := 60.0 // RRF 参数

	for _, results := range resultSets {
		for rank, result := range results {
			id := result.ID
			if id == "" {
				id = textutil.HashString(result.Content)
			}

			// RRF 分数：1 / (k + rank)
			scoreMap[id] += 1.0 / (k + float64(rank+1))
			if _, exists := resultMap[id]; !exists {
				resultMap[id] = result
			}
		}
	}

	// 转换为切片并排序
	merged := make([]SearchResult, 0, len(resultMap))
	for id, result := range resultMap {
		result.Score = float32(scoreMap[id])
		merged = append(merged, result)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	return merged
}
