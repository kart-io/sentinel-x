// Package evaluator 提供 RAG 评估功能，基于 Ragas 框架实现。
//
// 该包提供四个核心评估指标：
//   - Faithfulness（忠实度）: 衡量答案与上下文的事实一致性
//   - AnswerRelevancy（答案相关性）: 衡量答案与问题的相关程度
//   - ContextPrecision（上下文精确度）: 衡量检索上下文的相关性排名
//   - ContextRecall（上下文召回率）: 衡量检索内容覆盖理想答案的程度
//
// 使用示例:
//
//	evaluator := evaluator.New(chatProvider, embedProvider)
//	result, err := evaluator.Evaluate(ctx, &evaluator.Input{
//	    Question: "What is Milvus?",
//	    Answer:   "Milvus is a vector database.",
//	    Contexts: []string{"Milvus is an open-source vector database..."},
//	})
package evaluator

import (
	"context"
	"fmt"
	"strings"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/textutil"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// Input 评估输入参数。
type Input struct {
	// Question 用户问题。
	Question string `json:"question"`

	// Answer RAG 生成的答案。
	Answer string `json:"answer"`

	// Contexts 检索到的上下文列表。
	Contexts []string `json:"contexts"`

	// GroundTruth 期望的参考答案（可选，用于 Context Recall）。
	GroundTruth string `json:"ground_truth,omitempty"`
}

// Result 包含 RAG 评估结果。
type Result struct {
	// Faithfulness 忠实度评分 (0-1)，衡量答案与上下文的事实一致性。
	Faithfulness float64 `json:"faithfulness"`

	// AnswerRelevancy 答案相关性评分 (0-1)，衡量答案与问题的相关程度。
	AnswerRelevancy float64 `json:"answer_relevancy"`

	// ContextPrecision 上下文精确度评分 (0-1)，衡量检索上下文的相关性排名。
	ContextPrecision float64 `json:"context_precision"`

	// ContextRecall 上下文召回率评分 (0-1)，衡量检索内容覆盖理想答案的程度。
	ContextRecall float64 `json:"context_recall"`

	// OverallScore 综合评分 (0-1)，所有指标的加权平均。
	OverallScore float64 `json:"overall_score"`

	// Details 评估详情。
	Details *Details `json:"details,omitempty"`
}

// Details 包含评估的详细信息。
type Details struct {
	// Claims 从答案中提取的声明列表。
	Claims []string `json:"claims,omitempty"`

	// SupportedClaims 被上下文支持的声明数量。
	SupportedClaims int `json:"supported_claims,omitempty"`

	// TotalClaims 总声明数量。
	TotalClaims int `json:"total_claims,omitempty"`

	// RelevantContexts 相关上下文的索引列表。
	RelevantContexts []int `json:"relevant_contexts,omitempty"`

	// GeneratedQuestions 从答案生成的问题（用于答案相关性评估）。
	GeneratedQuestions []string `json:"generated_questions,omitempty"`
}

// WeightConfig 评估指标权重配置。
type WeightConfig struct {
	Faithfulness     float64
	AnswerRelevancy  float64
	ContextPrecision float64
	ContextRecall    float64
}

// DefaultWeights 返回默认的评估指标权重。
func DefaultWeights() WeightConfig {
	return WeightConfig{
		Faithfulness:     0.3,
		AnswerRelevancy:  0.3,
		ContextPrecision: 0.2,
		ContextRecall:    0.2,
	}
}

// Evaluator 提供 RAG 评估功能。
type Evaluator struct {
	chatProvider  llm.ChatProvider
	embedProvider llm.EmbeddingProvider
	weights       WeightConfig
}

// Option 配置 Evaluator 的选项。
type Option func(*Evaluator)

// WithWeights 设置评估指标权重。
func WithWeights(weights WeightConfig) Option {
	return func(e *Evaluator) {
		e.weights = weights
	}
}

// New 创建新的 RAG 评估器。
func New(chatProvider llm.ChatProvider, embedProvider llm.EmbeddingProvider, opts ...Option) *Evaluator {
	e := &Evaluator{
		chatProvider:  chatProvider,
		embedProvider: embedProvider,
		weights:       DefaultWeights(),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Evaluate 执行完整的 RAG 评估。
func (e *Evaluator) Evaluate(ctx context.Context, input *Input) (*Result, error) {
	result := &Result{
		Details: &Details{},
	}

	// 1. 评估忠实度
	faithfulness, claims, supportedCount, err := e.EvaluateFaithfulness(ctx, input.Answer, input.Contexts)
	if err != nil {
		logger.Warnf("Faithfulness evaluation failed: %v", err)
	} else {
		result.Faithfulness = faithfulness
		result.Details.Claims = claims
		result.Details.SupportedClaims = supportedCount
		result.Details.TotalClaims = len(claims)
	}

	// 2. 评估答案相关性
	relevancy, questions, err := e.EvaluateAnswerRelevancy(ctx, input.Answer, input.Question)
	if err != nil {
		logger.Warnf("Answer relevancy evaluation failed: %v", err)
	} else {
		result.AnswerRelevancy = relevancy
		result.Details.GeneratedQuestions = questions
	}

	// 3. 评估上下文精确度
	precision, relevantIdxs, err := e.EvaluateContextPrecision(ctx, input.Contexts, input.Question)
	if err != nil {
		logger.Warnf("Context precision evaluation failed: %v", err)
	} else {
		result.ContextPrecision = precision
		result.Details.RelevantContexts = relevantIdxs
	}

	// 4. 评估上下文召回率（需要参考答案）
	if input.GroundTruth != "" {
		recall, err := e.EvaluateContextRecall(ctx, input.Contexts, input.GroundTruth)
		if err != nil {
			logger.Warnf("Context recall evaluation failed: %v", err)
		} else {
			result.ContextRecall = recall
		}
	}

	// 5. 计算综合评分
	result.OverallScore = e.calculateOverallScore(result)

	return result, nil
}

// EvaluateFaithfulness 评估答案的忠实度。
// 忠实度衡量生成答案中的声明是否都能被检索上下文支持。
func (e *Evaluator) EvaluateFaithfulness(ctx context.Context, answer string, contexts []string) (float64, []string, int, error) {
	if answer == "" || len(contexts) == 0 {
		return 0, nil, 0, nil
	}

	// 步骤 1: 从答案中提取声明
	claims, err := e.extractClaims(ctx, answer)
	if err != nil {
		return 0, nil, 0, fmt.Errorf("提取声明失败: %w", err)
	}

	if len(claims) == 0 {
		return 1.0, nil, 0, nil // 无声明则认为忠实
	}

	// 步骤 2: 验证每个声明是否被上下文支持
	combinedContext := strings.Join(contexts, "\n\n")
	supportedCount := 0

	for _, claim := range claims {
		supported, err := e.verifyClaimAgainstContext(ctx, claim, combinedContext)
		if err != nil {
			logger.Warnf("验证声明失败: %v", err)
			continue
		}
		if supported {
			supportedCount++
		}
	}

	// 计算忠实度分数
	faithfulness := float64(supportedCount) / float64(len(claims))
	return faithfulness, claims, supportedCount, nil
}

// EvaluateAnswerRelevancy 评估答案与问题的相关性。
// 通过从答案生成问题，然后计算生成问题与原始问题的语义相似度。
func (e *Evaluator) EvaluateAnswerRelevancy(ctx context.Context, answer, question string) (float64, []string, error) {
	if answer == "" || question == "" {
		return 0, nil, nil
	}

	// 步骤 1: 从答案生成可能的问题
	generatedQuestions, err := e.generateQuestionsFromAnswer(ctx, answer, 3)
	if err != nil {
		return 0, nil, fmt.Errorf("生成问题失败: %w", err)
	}

	if len(generatedQuestions) == 0 {
		return 0, nil, nil
	}

	// 步骤 2: 计算原始问题与生成问题的语义相似度
	// 获取原始问题的嵌入
	questionEmbed, err := e.embedProvider.EmbedSingle(ctx, question)
	if err != nil {
		return 0, generatedQuestions, fmt.Errorf("嵌入原始问题失败: %w", err)
	}

	// 获取生成问题的嵌入
	genEmbeds, err := e.embedProvider.Embed(ctx, generatedQuestions)
	if err != nil {
		return 0, generatedQuestions, fmt.Errorf("嵌入生成问题失败: %w", err)
	}

	// 计算平均余弦相似度
	var totalSimilarity float64
	for _, genEmbed := range genEmbeds {
		similarity := textutil.CosineSimilarity(questionEmbed, genEmbed)
		totalSimilarity += similarity
	}

	avgSimilarity := totalSimilarity / float64(len(genEmbeds))
	// 归一化到 0-1 范围
	relevancy := textutil.NormalizeCosineSimilarity(avgSimilarity)

	return relevancy, generatedQuestions, nil
}

// EvaluateContextPrecision 评估上下文精确度。
// 衡量检索到的上下文中，相关内容的排名位置。
func (e *Evaluator) EvaluateContextPrecision(ctx context.Context, contexts []string, question string) (float64, []int, error) {
	if len(contexts) == 0 || question == "" {
		return 0, nil, nil
	}

	// 评估每个上下文是否与问题相关
	var relevantIdxs []int
	for i, context := range contexts {
		relevant, err := e.isContextRelevant(ctx, context, question)
		if err != nil {
			logger.Warnf("评估上下文相关性失败: %v", err)
			continue
		}
		if relevant {
			relevantIdxs = append(relevantIdxs, i)
		}
	}

	if len(relevantIdxs) == 0 {
		return 0, relevantIdxs, nil
	}

	// 计算加权累积精确度 (Weighted Cumulative Precision)
	// WCP = sum((precision@k * relevance@k)) / sum(relevance@k)
	var wcpSum float64
	var relevanceSum float64
	relevantCount := 0

	for k := 0; k < len(contexts); k++ {
		isRelevant := textutil.ContainsInt(relevantIdxs, k)
		if isRelevant {
			relevantCount++
			precisionAtK := float64(relevantCount) / float64(k+1)
			wcpSum += precisionAtK
			relevanceSum++
		}
	}

	if relevanceSum == 0 {
		return 0, relevantIdxs, nil
	}

	precision := wcpSum / relevanceSum
	return precision, relevantIdxs, nil
}

// EvaluateContextRecall 评估上下文召回率。
// 衡量检索的上下文是否包含生成理想答案所需的所有信息。
func (e *Evaluator) EvaluateContextRecall(ctx context.Context, contexts []string, groundTruth string) (float64, error) {
	if len(contexts) == 0 || groundTruth == "" {
		return 0, nil
	}

	// 从参考答案中提取声明
	claims, err := e.extractClaims(ctx, groundTruth)
	if err != nil {
		return 0, fmt.Errorf("从参考答案提取声明失败: %w", err)
	}

	if len(claims) == 0 {
		return 1.0, nil
	}

	// 检查每个声明是否可以从上下文中推导
	combinedContext := strings.Join(contexts, "\n\n")
	supportedCount := 0

	for _, claim := range claims {
		supported, err := e.verifyClaimAgainstContext(ctx, claim, combinedContext)
		if err != nil {
			continue
		}
		if supported {
			supportedCount++
		}
	}

	recall := float64(supportedCount) / float64(len(claims))
	return recall, nil
}

// extractClaims 从文本中提取原子性声明。
func (e *Evaluator) extractClaims(ctx context.Context, text string) ([]string, error) {
	prompt := fmt.Sprintf(`从以下文本中提取所有事实性声明。每个声明应该是独立的、可验证的单位。

文本:
%s

请以 JSON 数组格式返回声明列表，例如:
["声明1", "声明2", "声明3"]

只返回 JSON 数组，不要其他内容。`, text)

	response, err := e.chatProvider.Generate(ctx, prompt, "")
	if err != nil {
		return nil, err
	}

	// 解析 JSON 响应
	claims, err := textutil.ParseJSONArray(response)
	if err != nil {
		// 尝试按行分割
		claims = textutil.SplitByLines(response, 5)
	}

	return claims, nil
}

// verifyClaimAgainstContext 验证声明是否被上下文支持。
func (e *Evaluator) verifyClaimAgainstContext(ctx context.Context, claim, context string) (bool, error) {
	prompt := fmt.Sprintf(`判断以下声明是否被给定的上下文所支持或可以从中推导出来。

声明: %s

上下文:
%s

请只回答 "是" 或 "否"。`, claim, context)

	response, err := e.chatProvider.Generate(ctx, prompt, "")
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return strings.Contains(response, "是") || strings.Contains(response, "yes") || strings.Contains(response, "true"), nil
}

// generateQuestionsFromAnswer 从答案生成可能的问题。
func (e *Evaluator) generateQuestionsFromAnswer(ctx context.Context, answer string, count int) ([]string, error) {
	prompt := fmt.Sprintf(`根据以下答案，生成 %d 个可能导致这个答案的问题。

答案:
%s

请以 JSON 数组格式返回问题列表，例如:
["问题1?", "问题2?", "问题3?"]

只返回 JSON 数组，不要其他内容。`, count, answer)

	response, err := e.chatProvider.Generate(ctx, prompt, "")
	if err != nil {
		return nil, err
	}

	questions, err := textutil.ParseJSONArray(response)
	if err != nil {
		questions = textutil.SplitByLines(response, 5)
	}

	return questions, nil
}

// isContextRelevant 判断上下文是否与问题相关。
func (e *Evaluator) isContextRelevant(ctx context.Context, context, question string) (bool, error) {
	prompt := fmt.Sprintf(`判断以下上下文是否与给定的问题相关，即上下文是否包含回答问题所需的信息。

问题: %s

上下文:
%s

请只回答 "是" 或 "否"。`, question, context)

	response, err := e.chatProvider.Generate(ctx, prompt, "")
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return strings.Contains(response, "是") || strings.Contains(response, "yes") || strings.Contains(response, "true"), nil
}

// calculateOverallScore 计算综合评分。
func (e *Evaluator) calculateOverallScore(result *Result) float64 {
	var totalWeight float64
	var weightedSum float64

	if result.Faithfulness > 0 {
		weightedSum += result.Faithfulness * e.weights.Faithfulness
		totalWeight += e.weights.Faithfulness
	}
	if result.AnswerRelevancy > 0 {
		weightedSum += result.AnswerRelevancy * e.weights.AnswerRelevancy
		totalWeight += e.weights.AnswerRelevancy
	}
	if result.ContextPrecision > 0 {
		weightedSum += result.ContextPrecision * e.weights.ContextPrecision
		totalWeight += e.weights.ContextPrecision
	}
	if result.ContextRecall > 0 {
		weightedSum += result.ContextRecall * e.weights.ContextRecall
		totalWeight += e.weights.ContextRecall
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedSum / totalWeight
}
