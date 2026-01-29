package enhancer_test

import (
	"context"
	"testing"

	"github.com/kart-io/sentinel-x/internal/pkg/rag/enhancer"
	"github.com/kart-io/sentinel-x/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChatProvider 模拟聊天供应商，实现 llm.ChatProvider 接口。
type mockChatProvider struct{}

func (m *mockChatProvider) Generate(_ context.Context, prompt, _ string) (*llm.GenerateResponse, error) {
	var content string
	// 模拟查询重写
	switch {
	case containsSubstring(prompt, "查询优化专家"):
		content = "Milvus vector database features and capabilities"
	case containsSubstring(prompt, "假设性的答案"):
		// 模拟假设文档生成
		content = "Milvus is a powerful vector database that supports similarity search..."
	case containsSubstring(prompt, "评估以下文档与查询的相关性"):
		// 模拟相关性评分
		content = "0.85"
	default:
		content = "默认回复"
	}

	return &llm.GenerateResponse{
		Content: content,
		TokenUsage: &llm.TokenUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *mockChatProvider) Chat(_ context.Context, _ []llm.Message) (string, error) {
	return "模拟对话回复", nil
}

func (m *mockChatProvider) Name() string {
	return "mock-chat"
}

// mockEmbedProvider 模拟嵌入供应商，实现 llm.EmbeddingProvider 接口。
type mockEmbedProvider struct{}

func (m *mockEmbedProvider) EmbedSingle(_ context.Context, _ string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3, 0.4, 0.5}, nil
}

func (m *mockEmbedProvider) Embed(_ context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	}
	return result, nil
}

func (m *mockEmbedProvider) Name() string {
	return "mock-embed"
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestNewEnhancer(t *testing.T) {
	chatProvider := &mockChatProvider{}
	embedProvider := &mockEmbedProvider{}
	config := enhancer.DefaultConfig()

	e := enhancer.New(chatProvider, embedProvider, config)
	assert.NotNil(t, e)
}

func TestDefaultConfig(t *testing.T) {
	config := enhancer.DefaultConfig()

	assert.True(t, config.EnableQueryRewrite)
	assert.False(t, config.EnableHyDE)
	assert.True(t, config.EnableRerank)
	assert.True(t, config.EnableRepacking)
	assert.Equal(t, 5, config.RerankTopK)
}

func TestEnhanceQuery(t *testing.T) {
	chatProvider := &mockChatProvider{}
	embedProvider := &mockEmbedProvider{}
	config := enhancer.DefaultConfig()
	e := enhancer.New(chatProvider, embedProvider, config)

	ctx := context.Background()

	t.Run("启用查询重写", func(t *testing.T) {
		query, embeddings, err := e.EnhanceQuery(ctx, "What is Milvus?")
		require.NoError(t, err)
		assert.NotEmpty(t, query)
		assert.NotEmpty(t, embeddings)
		assert.Len(t, embeddings, 1) // 只有查询嵌入，未启用 HyDE
	})

	t.Run("启用 HyDE", func(t *testing.T) {
		hydeConfig := enhancer.DefaultConfig()
		hydeConfig.EnableHyDE = true
		hydeEnhancer := enhancer.New(chatProvider, embedProvider, hydeConfig)

		query, embeddings, err := hydeEnhancer.EnhanceQuery(ctx, "What is Milvus?")
		require.NoError(t, err)
		assert.NotEmpty(t, query)
		assert.Len(t, embeddings, 2) // 查询嵌入 + HyDE 嵌入
	})

	t.Run("禁用所有增强", func(t *testing.T) {
		disabledConfig := enhancer.Config{
			EnableQueryRewrite: false,
			EnableHyDE:         false,
			EnableRerank:       false,
			EnableRepacking:    false,
		}
		disabledEnhancer := enhancer.New(chatProvider, embedProvider, disabledConfig)

		query, embeddings, err := disabledEnhancer.EnhanceQuery(ctx, "What is Milvus?")
		require.NoError(t, err)
		assert.Equal(t, "What is Milvus?", query) // 原始查询不变
		assert.Len(t, embeddings, 1)
	})
}

func TestRerankResults(t *testing.T) {
	chatProvider := &mockChatProvider{}
	embedProvider := &mockEmbedProvider{}
	config := enhancer.DefaultConfig()
	e := enhancer.New(chatProvider, embedProvider, config)

	ctx := context.Background()

	t.Run("正常重排序", func(t *testing.T) {
		results := []enhancer.SearchResult{
			{ID: "1", Content: "文档1", Score: 0.5, Metadata: map[string]any{}},
			{ID: "2", Content: "文档2", Score: 0.6, Metadata: map[string]any{}},
			{ID: "3", Content: "文档3", Score: 0.7, Metadata: map[string]any{}},
		}

		reranked, err := e.RerankResults(ctx, "查询", results)
		require.NoError(t, err)
		assert.NotEmpty(t, reranked)
	})

	t.Run("空结果", func(t *testing.T) {
		reranked, err := e.RerankResults(ctx, "查询", []enhancer.SearchResult{})
		require.NoError(t, err)
		assert.Empty(t, reranked)
	})

	t.Run("禁用重排序", func(t *testing.T) {
		disabledConfig := enhancer.DefaultConfig()
		disabledConfig.EnableRerank = false
		disabledEnhancer := enhancer.New(chatProvider, embedProvider, disabledConfig)

		results := []enhancer.SearchResult{
			{ID: "1", Content: "文档1", Score: 0.5, Metadata: map[string]any{}},
		}

		reranked, err := disabledEnhancer.RerankResults(ctx, "查询", results)
		require.NoError(t, err)
		assert.Equal(t, results, reranked) // 返回原始结果
	})

	t.Run("TopK 截取", func(t *testing.T) {
		config := enhancer.DefaultConfig()
		config.RerankTopK = 2
		e := enhancer.New(chatProvider, embedProvider, config)

		results := []enhancer.SearchResult{
			{ID: "1", Content: "文档1", Score: 0.5, Metadata: map[string]any{}},
			{ID: "2", Content: "文档2", Score: 0.6, Metadata: map[string]any{}},
			{ID: "3", Content: "文档3", Score: 0.7, Metadata: map[string]any{}},
			{ID: "4", Content: "文档4", Score: 0.8, Metadata: map[string]any{}},
		}

		reranked, err := e.RerankResults(ctx, "查询", results)
		require.NoError(t, err)
		assert.Len(t, reranked, 2) // 只保留 TopK 个
	})
}

func TestRepackDocuments(t *testing.T) {
	chatProvider := &mockChatProvider{}
	embedProvider := &mockEmbedProvider{}
	config := enhancer.DefaultConfig()
	e := enhancer.New(chatProvider, embedProvider, config)

	t.Run("正常重组", func(t *testing.T) {
		results := []enhancer.SearchResult{
			{ID: "1", Content: "文档1", Score: 0.9, Metadata: map[string]any{}},
			{ID: "2", Content: "文档2", Score: 0.7, Metadata: map[string]any{}},
			{ID: "3", Content: "文档3", Score: 0.5, Metadata: map[string]any{}},
			{ID: "4", Content: "文档4", Score: 0.3, Metadata: map[string]any{}},
		}

		repacked := e.RepackDocuments(results)
		assert.Len(t, repacked, 4)
		// 验证高分文档在首尾位置
		assert.True(t, repacked[0].Score >= repacked[1].Score || repacked[0].Score >= repacked[2].Score)
	})

	t.Run("少于3个结果不重组", func(t *testing.T) {
		results := []enhancer.SearchResult{
			{ID: "1", Content: "文档1", Score: 0.9, Metadata: map[string]any{}},
			{ID: "2", Content: "文档2", Score: 0.7, Metadata: map[string]any{}},
		}

		repacked := e.RepackDocuments(results)
		assert.Equal(t, results, repacked) // 不变
	})

	t.Run("禁用重组", func(t *testing.T) {
		disabledConfig := enhancer.DefaultConfig()
		disabledConfig.EnableRepacking = false
		disabledEnhancer := enhancer.New(chatProvider, embedProvider, disabledConfig)

		results := []enhancer.SearchResult{
			{ID: "1", Content: "文档1", Score: 0.9, Metadata: map[string]any{}},
			{ID: "2", Content: "文档2", Score: 0.7, Metadata: map[string]any{}},
			{ID: "3", Content: "文档3", Score: 0.5, Metadata: map[string]any{}},
		}

		repacked := disabledEnhancer.RepackDocuments(results)
		assert.Equal(t, results, repacked) // 返回原始结果
	})
}

func TestMergeEmbeddingResults(t *testing.T) {
	t.Run("单结果集", func(t *testing.T) {
		results := []enhancer.SearchResult{
			{ID: "1", Content: "文档1", Score: 0.9, Metadata: map[string]any{}},
		}

		merged := enhancer.MergeEmbeddingResults([][]enhancer.SearchResult{results})
		assert.Equal(t, results, merged)
	})

	t.Run("多结果集合并", func(t *testing.T) {
		set1 := []enhancer.SearchResult{
			{ID: "1", Content: "文档1", Score: 0.9, Metadata: map[string]any{}},
			{ID: "2", Content: "文档2", Score: 0.7, Metadata: map[string]any{}},
		}
		set2 := []enhancer.SearchResult{
			{ID: "2", Content: "文档2", Score: 0.8, Metadata: map[string]any{}},
			{ID: "3", Content: "文档3", Score: 0.6, Metadata: map[string]any{}},
		}

		merged := enhancer.MergeEmbeddingResults([][]enhancer.SearchResult{set1, set2})
		assert.NotEmpty(t, merged)
		// 验证去重：ID "2" 只出现一次
		idCount := make(map[string]int)
		for _, r := range merged {
			idCount[r.ID]++
		}
		assert.Equal(t, 1, idCount["2"])
	})

	t.Run("空结果集", func(t *testing.T) {
		merged := enhancer.MergeEmbeddingResults([][]enhancer.SearchResult{})
		assert.Nil(t, merged)
	})
}

func TestSearchResultStruct(t *testing.T) {
	result := enhancer.SearchResult{
		ID:      "test-id",
		Content: "测试内容",
		Score:   0.85,
		Metadata: map[string]any{
			"document_name": "test.md",
			"section":       "Introduction",
		},
	}

	assert.Equal(t, "test-id", result.ID)
	assert.Equal(t, "测试内容", result.Content)
	assert.Equal(t, float32(0.85), result.Score)
	assert.Equal(t, "test.md", result.Metadata["document_name"])
}

func TestConfigStruct(t *testing.T) {
	config := enhancer.Config{
		EnableQueryRewrite: true,
		EnableHyDE:         true,
		EnableRerank:       true,
		EnableRepacking:    true,
		RerankTopK:         10,
	}

	assert.True(t, config.EnableQueryRewrite)
	assert.True(t, config.EnableHyDE)
	assert.True(t, config.EnableRerank)
	assert.True(t, config.EnableRepacking)
	assert.Equal(t, 10, config.RerankTopK)
}
