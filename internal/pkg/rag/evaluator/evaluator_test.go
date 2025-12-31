package evaluator_test

import (
	"context"
	"testing"

	"github.com/kart-io/sentinel-x/internal/pkg/rag/evaluator"
	"github.com/kart-io/sentinel-x/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChatProvider 模拟聊天供应商，实现 llm.ChatProvider 接口。
type mockChatProvider struct {
	responses map[string]string
}

func (m *mockChatProvider) Generate(ctx context.Context, prompt, systemPrompt string) (string, error) {
	// 简单的模式匹配返回
	if containsSubstring(prompt, "提取所有事实性声明") {
		return `["这是声明1", "这是声明2"]`, nil
	}
	if containsSubstring(prompt, "是否被给定的上下文所支持") {
		return "是", nil
	}
	if containsSubstring(prompt, "生成") && containsSubstring(prompt, "问题") {
		return `["问题1?", "问题2?", "问题3?"]`, nil
	}
	if containsSubstring(prompt, "是否与给定的问题相关") {
		return "是", nil
	}
	return "默认回复", nil
}

func (m *mockChatProvider) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return "模拟对话回复", nil
}

func (m *mockChatProvider) Name() string {
	return "mock-chat"
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mockEmbedProvider 模拟嵌入供应商，实现 llm.EmbeddingProvider 接口。
type mockEmbedProvider struct{}

func (m *mockEmbedProvider) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	// 返回固定的嵌入向量
	return []float32{0.1, 0.2, 0.3, 0.4, 0.5}, nil
}

func (m *mockEmbedProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	}
	return result, nil
}

func (m *mockEmbedProvider) Name() string {
	return "mock-embed"
}

func TestNewEvaluator(t *testing.T) {
	chatProvider := &mockChatProvider{}
	embedProvider := &mockEmbedProvider{}

	eval := evaluator.New(chatProvider, embedProvider)
	assert.NotNil(t, eval)
}

func TestNewEvaluatorWithWeights(t *testing.T) {
	chatProvider := &mockChatProvider{}
	embedProvider := &mockEmbedProvider{}

	weights := evaluator.WeightConfig{
		Faithfulness:     0.4,
		AnswerRelevancy:  0.4,
		ContextPrecision: 0.1,
		ContextRecall:    0.1,
	}

	eval := evaluator.New(chatProvider, embedProvider, evaluator.WithWeights(weights))
	assert.NotNil(t, eval)
}

func TestDefaultWeights(t *testing.T) {
	weights := evaluator.DefaultWeights()

	assert.Equal(t, 0.3, weights.Faithfulness)
	assert.Equal(t, 0.3, weights.AnswerRelevancy)
	assert.Equal(t, 0.2, weights.ContextPrecision)
	assert.Equal(t, 0.2, weights.ContextRecall)
}

func TestEvaluate(t *testing.T) {
	chatProvider := &mockChatProvider{}
	embedProvider := &mockEmbedProvider{}
	eval := evaluator.New(chatProvider, embedProvider)

	ctx := context.Background()
	input := &evaluator.Input{
		Question: "什么是 Milvus?",
		Answer:   "Milvus 是一个向量数据库。",
		Contexts: []string{
			"Milvus 是一个开源的向量数据库。",
			"它支持大规模向量检索。",
		},
	}

	result, err := eval.Evaluate(ctx, input)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// 验证结果结构
	assert.GreaterOrEqual(t, result.Faithfulness, 0.0)
	assert.LessOrEqual(t, result.Faithfulness, 1.0)
	assert.GreaterOrEqual(t, result.AnswerRelevancy, 0.0)
	assert.LessOrEqual(t, result.AnswerRelevancy, 1.0)
	assert.GreaterOrEqual(t, result.OverallScore, 0.0)
	assert.LessOrEqual(t, result.OverallScore, 1.0)
	assert.NotNil(t, result.Details)
}

func TestEvaluateFaithfulness(t *testing.T) {
	chatProvider := &mockChatProvider{}
	embedProvider := &mockEmbedProvider{}
	eval := evaluator.New(chatProvider, embedProvider)

	ctx := context.Background()

	t.Run("正常评估", func(t *testing.T) {
		score, claims, supported, err := eval.EvaluateFaithfulness(ctx, "这是答案", []string{"这是上下文"})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, score, 0.0)
		assert.LessOrEqual(t, score, 1.0)
		assert.NotEmpty(t, claims)
		assert.GreaterOrEqual(t, supported, 0)
	})

	t.Run("空答案", func(t *testing.T) {
		score, claims, supported, err := eval.EvaluateFaithfulness(ctx, "", []string{"上下文"})
		require.NoError(t, err)
		assert.Equal(t, 0.0, score)
		assert.Nil(t, claims)
		assert.Equal(t, 0, supported)
	})

	t.Run("空上下文", func(t *testing.T) {
		score, _, _, err := eval.EvaluateFaithfulness(ctx, "答案", []string{})
		require.NoError(t, err)
		assert.Equal(t, 0.0, score)
	})
}

func TestEvaluateAnswerRelevancy(t *testing.T) {
	chatProvider := &mockChatProvider{}
	embedProvider := &mockEmbedProvider{}
	eval := evaluator.New(chatProvider, embedProvider)

	ctx := context.Background()

	t.Run("正常评估", func(t *testing.T) {
		score, questions, err := eval.EvaluateAnswerRelevancy(ctx, "这是答案", "这是问题?")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, score, 0.0)
		assert.LessOrEqual(t, score, 1.0)
		assert.NotEmpty(t, questions)
	})

	t.Run("空答案", func(t *testing.T) {
		score, _, err := eval.EvaluateAnswerRelevancy(ctx, "", "问题")
		require.NoError(t, err)
		assert.Equal(t, 0.0, score)
	})

	t.Run("空问题", func(t *testing.T) {
		score, _, err := eval.EvaluateAnswerRelevancy(ctx, "答案", "")
		require.NoError(t, err)
		assert.Equal(t, 0.0, score)
	})
}

func TestEvaluateContextPrecision(t *testing.T) {
	chatProvider := &mockChatProvider{}
	embedProvider := &mockEmbedProvider{}
	eval := evaluator.New(chatProvider, embedProvider)

	ctx := context.Background()

	t.Run("正常评估", func(t *testing.T) {
		contexts := []string{"相关上下文1", "相关上下文2"}
		score, relevantIdxs, err := eval.EvaluateContextPrecision(ctx, contexts, "问题?")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, score, 0.0)
		assert.LessOrEqual(t, score, 1.0)
		assert.NotEmpty(t, relevantIdxs)
	})

	t.Run("空上下文", func(t *testing.T) {
		score, _, err := eval.EvaluateContextPrecision(ctx, []string{}, "问题")
		require.NoError(t, err)
		assert.Equal(t, 0.0, score)
	})

	t.Run("空问题", func(t *testing.T) {
		score, _, err := eval.EvaluateContextPrecision(ctx, []string{"上下文"}, "")
		require.NoError(t, err)
		assert.Equal(t, 0.0, score)
	})
}

func TestEvaluateContextRecall(t *testing.T) {
	chatProvider := &mockChatProvider{}
	embedProvider := &mockEmbedProvider{}
	eval := evaluator.New(chatProvider, embedProvider)

	ctx := context.Background()

	t.Run("正常评估", func(t *testing.T) {
		contexts := []string{"上下文1", "上下文2"}
		score, err := eval.EvaluateContextRecall(ctx, contexts, "参考答案")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, score, 0.0)
		assert.LessOrEqual(t, score, 1.0)
	})

	t.Run("空上下文", func(t *testing.T) {
		score, err := eval.EvaluateContextRecall(ctx, []string{}, "参考答案")
		require.NoError(t, err)
		assert.Equal(t, 0.0, score)
	})

	t.Run("空参考答案", func(t *testing.T) {
		score, err := eval.EvaluateContextRecall(ctx, []string{"上下文"}, "")
		require.NoError(t, err)
		assert.Equal(t, 0.0, score)
	})
}

func TestInputStruct(t *testing.T) {
	input := &evaluator.Input{
		Question:    "测试问题",
		Answer:      "测试答案",
		Contexts:    []string{"上下文1", "上下文2"},
		GroundTruth: "参考答案",
	}

	assert.Equal(t, "测试问题", input.Question)
	assert.Equal(t, "测试答案", input.Answer)
	assert.Len(t, input.Contexts, 2)
	assert.Equal(t, "参考答案", input.GroundTruth)
}

func TestResultStruct(t *testing.T) {
	result := &evaluator.Result{
		Faithfulness:     0.9,
		AnswerRelevancy:  0.85,
		ContextPrecision: 0.8,
		ContextRecall:    0.75,
		OverallScore:     0.825,
		Details: &evaluator.Details{
			Claims:             []string{"声明1", "声明2"},
			SupportedClaims:    2,
			TotalClaims:        2,
			RelevantContexts:   []int{0, 1},
			GeneratedQuestions: []string{"问题1", "问题2"},
		},
	}

	assert.Equal(t, 0.9, result.Faithfulness)
	assert.Equal(t, 0.85, result.AnswerRelevancy)
	assert.NotNil(t, result.Details)
	assert.Len(t, result.Details.Claims, 2)
}
