package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/kart-io/sentinel-x/internal/rag/store"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// 测试常量
const (
	mockEmbeddingProviderName = "mock-embedding"
	testDocumentID            = "doc_test"
)

// === Mock 实现 ===

// mockVectorStoreForService 模拟 VectorStore 用于 RAGService 集成测试。
type mockVectorStoreForService struct {
	searchResults      []*store.SearchResult
	searchError        error
	shouldReturnEmpty  bool
	searchWithFilterFn func(ctx context.Context, collection string, embedding []float32, expr string, topK int) ([]*store.SearchResult, error)
	searchCallCount    int // 调用计数器
}

func (m *mockVectorStoreForService) SearchWithFilter(ctx context.Context, collection string, embedding []float32, expr string, topK int) ([]*store.SearchResult, error) {
	if m.searchWithFilterFn != nil {
		return m.searchWithFilterFn(ctx, collection, embedding, expr, topK)
	}

	if m.searchError != nil {
		return nil, m.searchError
	}

	if m.shouldReturnEmpty {
		return []*store.SearchResult{}, nil
	}

	return m.searchResults, nil
}

func (m *mockVectorStoreForService) Search(ctx context.Context, collection string, embedding []float32, topK int) ([]*store.SearchResult, error) {
	m.searchCallCount++ // 递增计数器

	if m.searchError != nil {
		return nil, m.searchError
	}

	if m.shouldReturnEmpty {
		return []*store.SearchResult{}, nil
	}

	return m.searchResults, nil
}

func (m *mockVectorStoreForService) Insert(ctx context.Context, collection string, chunks []*store.Chunk) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (m *mockVectorStoreForService) CreateCollection(ctx context.Context, config *store.CollectionConfig) error {
	return errors.New("not implemented")
}

func (m *mockVectorStoreForService) GetStats(ctx context.Context, collection string) (int64, error) {
	return 100, nil // 返回固定值
}

func (m *mockVectorStoreForService) Close(ctx context.Context) error {
	return nil
}

var _ store.VectorStore = (*mockVectorStoreForService)(nil)

// mockEmbeddingProviderForService 模拟 EmbeddingProvider。
type mockEmbeddingProviderForService struct {
	embedding []float32
	err       error
}

func (m *mockEmbeddingProviderForService) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = m.embedding
	}
	return result, nil
}

func (m *mockEmbeddingProviderForService) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.embedding, nil
}

func (m *mockEmbeddingProviderForService) Name() string {
	return mockEmbeddingProviderName
}

var _ llm.EmbeddingProvider = (*mockEmbeddingProviderForService)(nil)

// mockChatProviderForService 模拟 ChatProvider。
type mockChatProviderForService struct {
	response *llm.GenerateResponse
	err      error
}

func (m *mockChatProviderForService) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "这是对话响应", nil
}

func (m *mockChatProviderForService) Generate(ctx context.Context, prompt string, systemPrompt string) (*llm.GenerateResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.response != nil {
		return m.response, nil
	}
	// 默认响应
	return &llm.GenerateResponse{
		Content: "这是生成的答案",
		TokenUsage: &llm.TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
		},
	}, nil
}

func (m *mockChatProviderForService) Name() string {
	return "mock-chat"
}

var _ llm.ChatProvider = (*mockChatProviderForService)(nil)

// === 测试用例 ===

// TestRAGService_Query_TreeDisabled 测试树检索禁用时的场景（向量检索）。
func TestRAGService_Query_TreeDisabled(t *testing.T) {
	// 准备 Mock 数据
	mockStore := &mockVectorStoreForService{
		searchResults: []*store.SearchResult{
			{
				ID:           "chunk_1",
				DocumentID:   testDocumentID,
				DocumentName: "测试文档",
				Section:      "第一章",
				Content:      "这是向量检索的结果",
				Score:        0.9,
			},
		},
	}

	mockEmbed := &mockEmbeddingProviderForService{
		embedding: make([]float32, 768),
	}

	mockChat := &mockChatProviderForService{}

	// 创建服务配置（树功能禁用）
	config := &ServiceConfig{
		IndexerConfig: &IndexerConfig{
			Collection: "test_collection",
			ChunkSize:  500,
		},
		RetrieverConfig: &RetrieverConfig{
			Collection: "test_collection",
			TopK:       10,
		},
		GeneratorConfig: &GeneratorConfig{
			SystemPrompt: "你是一个AI助手",
		},
		QueryCacheConfig: nil,   // 禁用缓存
		TreeEnabled:      false, // 关键：树功能禁用
	}

	// 创建 RAGService
	service := NewRAGService(mockStore, mockEmbed, mockChat, nil, config)

	// 验证树组件未初始化
	if service.treeRetriever != nil {
		t.Error("TreeEnabled=false 时 treeRetriever 应为 nil")
	}
	if service.treeBuilder != nil {
		t.Error("TreeEnabled=false 时 treeBuilder 应为 nil")
	}
	if service.treeEnabled {
		t.Error("TreeEnabled=false 时 treeEnabled 应为 false")
	}

	// 执行查询
	result, err := service.Query(context.Background(), "测试问题")
	if err != nil {
		t.Fatalf("Query() 返回错误: %v", err)
	}

	// 验证结果
	if result == nil {
		t.Fatal("Query() 返回 nil 结果")
	}

	if result.Answer != "这是生成的答案" {
		t.Errorf("Answer = %q, 期望 = %q", result.Answer, "这是生成的答案")
	}

	if len(result.Sources) != 1 {
		t.Errorf("Sources 长度 = %d, 期望 = 1", len(result.Sources))
	}

	if len(result.Sources) > 0 {
		source := result.Sources[0]
		if source.DocumentID != testDocumentID {
			t.Errorf("Source.DocumentID = %q, 期望 = %q", source.DocumentID, testDocumentID)
		}
		if source.Content != "这是向量检索的结果" {
			t.Errorf("Source.Content = %q, 期望包含向量检索结果", source.Content)
		}
	}

	t.Log("✅ TreeEnabled=false 场景测试通过：向量检索正常工作")
}

// TestRAGService_Query_TreeEnabled_Success 测试树检索启用且成功的场景。
func TestRAGService_Query_TreeEnabled_Success(t *testing.T) {
	// 准备 Mock 数据：模拟树结构
	mockStore := &mockVectorStoreForService{
		searchWithFilterFn: func(ctx context.Context, collection string, embedding []float32, expr string, topK int) ([]*store.SearchResult, error) {
			// 根据过滤条件返回不同结果
			if contains(expr, "node_type == 2") {
				// 根节点查询
				return []*store.SearchResult{
					{
						ID:           "root_1",
						DocumentID:   testDocumentID,
						DocumentName: "测试文档",
						Content:      "根节点摘要",
						Score:        0.9,
					},
				}, nil
			} else if contains(expr, "level == 0") {
				// 叶子节点查询
				return []*store.SearchResult{
					{
						ID:           "leaf_1",
						DocumentID:   testDocumentID,
						DocumentName: "测试文档",
						Content:      "这是树检索的叶子节点内容",
						Score:        0.85,
					},
				}, nil
			}
			return []*store.SearchResult{}, nil
		},
	}

	mockEmbed := &mockEmbeddingProviderForService{
		embedding: make([]float32, 768),
	}

	mockChat := &mockChatProviderForService{}

	// 创建服务配置（树功能启用）
	config := &ServiceConfig{
		IndexerConfig: &IndexerConfig{
			Collection: "test_collection",
			ChunkSize:  500,
		},
		RetrieverConfig: &RetrieverConfig{
			Collection: "test_collection",
			TopK:       10,
		},
		GeneratorConfig: &GeneratorConfig{
			SystemPrompt: "你是一个AI助手",
		},
		QueryCacheConfig: nil, // 禁用缓存
		TreeRetrieverConfig: &TreeRetrieverConfig{
			Collection:       "test_collection",
			TopKPath:         3,
			TopKLeaf:         20,
			ScoreWeightSim:   0.7,
			ScoreWeightLevel: 0.3,
			MaxLevel:         2,
		},
		TreeBuilderConfig: &TreeBuilderConfig{
			Collection:  "test_collection",
			MaxLevel:    2,
			NumClusters: 5,
		},
		TreeEnabled: true, // 关键：树功能启用
	}

	// 创建 RAGService
	service := NewRAGService(mockStore, mockEmbed, mockChat, nil, config)

	// 验证树组件已初始化
	if service.treeRetriever == nil {
		t.Fatal("TreeEnabled=true 时 treeRetriever 不应为 nil")
	}
	if service.treeBuilder == nil {
		t.Fatal("TreeEnabled=true 时 treeBuilder 不应为 nil")
	}
	if !service.treeEnabled {
		t.Error("TreeEnabled=true 时 treeEnabled 应为 true")
	}

	// 执行查询
	result, err := service.Query(context.Background(), "测试问题")
	if err != nil {
		t.Fatalf("Query() 返回错误: %v", err)
	}

	// 验证结果
	if result == nil {
		t.Fatal("Query() 返回 nil 结果")
	}

	if result.Answer != "这是生成的答案" {
		t.Errorf("Answer = %q, 期望 = %q", result.Answer, "这是生成的答案")
	}

	// 验证使用了树检索的结果
	if len(result.Sources) > 0 {
		hasTreeResult := false
		for _, source := range result.Sources {
			if source.Content == "这是树检索的叶子节点内容" {
				hasTreeResult = true
				break
			}
		}
		if !hasTreeResult {
			t.Error("结果中应包含树检索的内容")
		}
	}

	t.Log("✅ TreeEnabled=true 场景测试通过：树检索正常工作")
}

// TestRAGService_Query_TreeEnabled_Fallback 测试树检索失败时的降级场景。
func TestRAGService_Query_TreeEnabled_Fallback(t *testing.T) {
	// 准备 Mock 数据：模拟树检索失败
	treeCallCount := 0

	mockStore := &mockVectorStoreForService{
		searchWithFilterFn: func(ctx context.Context, collection string, embedding []float32, expr string, topK int) ([]*store.SearchResult, error) {
			treeCallCount++
			// 树检索失败
			return nil, errors.New("树索引不存在")
		},
		searchResults: []*store.SearchResult{
			{
				ID:           "chunk_1",
				DocumentID:   testDocumentID,
				DocumentName: "测试文档",
				Content:      "这是向量检索的降级结果",
				Score:        0.8,
			},
		},
	}

	mockEmbed := &mockEmbeddingProviderForService{
		embedding: make([]float32, 768),
	}

	mockChat := &mockChatProviderForService{}

	// 创建服务配置（树功能启用）
	config := &ServiceConfig{
		IndexerConfig: &IndexerConfig{
			Collection: "test_collection",
			ChunkSize:  500,
		},
		RetrieverConfig: &RetrieverConfig{
			Collection: "test_collection",
			TopK:       10,
		},
		GeneratorConfig: &GeneratorConfig{
			SystemPrompt: "你是一个AI助手",
		},
		QueryCacheConfig: nil,
		TreeRetrieverConfig: &TreeRetrieverConfig{
			Collection:       "test_collection",
			TopKPath:         3,
			TopKLeaf:         20,
			ScoreWeightSim:   0.7,
			ScoreWeightLevel: 0.3,
			MaxLevel:         2,
		},
		TreeBuilderConfig: &TreeBuilderConfig{
			Collection:  "test_collection",
			MaxLevel:    2,
			NumClusters: 5,
		},
		TreeEnabled: true,
	}

	// 创建 RAGService
	service := NewRAGService(mockStore, mockEmbed, mockChat, nil, config)

	// 执行查询
	result, err := service.Query(context.Background(), "测试问题")
	if err != nil {
		t.Fatalf("Query() 返回错误: %v", err)
	}

	// 验证结果
	if result == nil {
		t.Fatal("Query() 返回 nil 结果")
	}

	// 验证降级到了向量检索
	if mockStore.searchCallCount == 0 {
		t.Error("树检索失败后应该调用向量检索（降级）")
	}

	// 验证使用了降级结果
	if len(result.Sources) > 0 {
		source := result.Sources[0]
		if source.Content != "这是向量检索的降级结果" {
			t.Errorf("应使用向量检索的降级结果，实际 Content = %q", source.Content)
		}
	}

	t.Logf("✅ 降级测试通过：树检索失败后成功降级到向量检索（树调用=%d次，向量调用=%d次）", treeCallCount, mockStore.searchCallCount)
}

// TestRAGService_GetStats 测试统计信息获取。
func TestRAGService_GetStats(t *testing.T) {
	mockStore := &mockVectorStoreForService{
		searchResults: []*store.SearchResult{},
	}

	mockEmbed := &mockEmbeddingProviderForService{
		embedding: make([]float32, 768),
	}

	mockChat := &mockChatProviderForService{}

	config := &ServiceConfig{
		IndexerConfig: &IndexerConfig{
			Collection: "test_collection",
			ChunkSize:  500,
		},
		RetrieverConfig: &RetrieverConfig{
			Collection: "test_collection",
			TopK:       10,
		},
		GeneratorConfig: &GeneratorConfig{
			SystemPrompt: "你是一个AI助手",
		},
		QueryCacheConfig: nil,
		TreeEnabled:      false,
	}

	service := NewRAGService(mockStore, mockEmbed, mockChat, nil, config)

	// 获取统计信息
	stats, err := service.GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats() 返回错误: %v", err)
	}

	// 验证基本字段
	if stats["collection"] != "test_collection" {
		t.Errorf("collection = %v, 期望 = %q", stats["collection"], "test_collection")
	}

	if stats["chunk_count"] != int64(100) {
		t.Errorf("chunk_count = %v, 期望 = 100", stats["chunk_count"])
	}

	if stats["embed_provider"] != mockEmbeddingProviderName {
		t.Errorf("embed_provider = %v, 期望 = %q", stats["embed_provider"], "mock-embedding")
	}

	if stats["chat_provider"] != "mock-chat" {
		t.Errorf("chat_provider = %v, 期望 = %q", stats["chat_provider"], "mock-chat")
	}

	// 验证 metrics 字段存在
	if stats["metrics"] == nil {
		t.Error("metrics 字段不应为 nil")
	}

	t.Log("✅ GetStats 测试通过")
}
