package biz

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/kart-io/sentinel-x/internal/rag/store"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// === Mock 实现 ===

// mockVectorStoreForRetriever 模拟 VectorStore 用于 TreeRetriever 测试。
type mockVectorStoreForRetriever struct {
	searchResults     []*store.SearchResult
	searchError       error
	shouldReturnEmpty bool
}

func (m *mockVectorStoreForRetriever) SearchWithFilter(ctx context.Context, collection string, embedding []float32, expr string, topK int) ([]*store.SearchResult, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}

	if m.shouldReturnEmpty {
		return []*store.SearchResult{}, nil
	}

	return m.searchResults, nil
}

func (m *mockVectorStoreForRetriever) Search(ctx context.Context, collection string, embedding []float32, topK int) ([]*store.SearchResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockVectorStoreForRetriever) Insert(ctx context.Context, collection string, chunks []*store.Chunk) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (m *mockVectorStoreForRetriever) CreateCollection(ctx context.Context, config *store.CollectionConfig) error {
	return errors.New("not implemented")
}

func (m *mockVectorStoreForRetriever) GetStats(ctx context.Context, collection string) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *mockVectorStoreForRetriever) Close(ctx context.Context) error {
	return nil
}

var _ store.VectorStore = (*mockVectorStoreForRetriever)(nil)

// mockEmbeddingProviderForRetriever 模拟 EmbeddingProvider 用于 TreeRetriever 测试。
type mockEmbeddingProviderForRetriever struct {
	embedding []float32
	err       error
}

func (m *mockEmbeddingProviderForRetriever) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = m.embedding
	}
	return result, nil
}

func (m *mockEmbeddingProviderForRetriever) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.embedding, nil
}

func (m *mockEmbeddingProviderForRetriever) Name() string {
	return "mock-embedding"
}

var _ llm.EmbeddingProvider = (*mockEmbeddingProviderForRetriever)(nil)

// === 测试用例 ===

// TestTreeRetriever_Retrieve_Success 测试成功的三阶段检索。
func TestTreeRetriever_Retrieve_Success(t *testing.T) {
	// 准备测试数据：根节点、叶子节点
	mockStore := &mockVectorStoreForRetriever{
		searchResults: []*store.SearchResult{
			{
				ID:           "root_1",
				DocumentID:   "doc_test",
				DocumentName: "测试文档",
				Section:      "section_1",
				Content:      "根节点内容",
				Score:        0.9,
			},
			{
				ID:           "leaf_1",
				DocumentID:   "doc_test",
				DocumentName: "测试文档",
				Section:      "section_1",
				Content:      "叶子节点内容",
				Score:        0.85,
			},
		},
	}

	mockEmbed := &mockEmbeddingProviderForRetriever{
		embedding: make([]float32, 768),
	}

	config := &TreeRetrieverConfig{
		Collection:       "test_collection",
		TopKPath:         3,
		TopKLeaf:         20,
		ScoreWeightSim:   0.7,
		ScoreWeightLevel: 0.3,
		MaxLevel:         2,
	}

	retriever := NewTreeRetriever(mockStore, mockEmbed, config)

	// 执行检索
	result, err := retriever.Retrieve(context.Background(), "测试问题", "doc_test")
	if err != nil {
		t.Fatalf("Retrieve() 返回错误: %v", err)
	}

	// 验证结果
	if result == nil {
		t.Fatal("Retrieve() 返回 nil 结果")
	}

	if result.Query != "测试问题" {
		t.Errorf("结果 Query = %q, 期望 = %q", result.Query, "测试问题")
	}

	if len(result.Results) == 0 {
		t.Error("Retrieve() 返回空结果")
	}

	t.Logf("检索到 %d 个结果", len(result.Results))
}

// TestTreeRetriever_Retrieve_EmptyResults 测试空结果处理。
func TestTreeRetriever_Retrieve_EmptyResults(t *testing.T) {
	mockStore := &mockVectorStoreForRetriever{
		shouldReturnEmpty: true,
	}

	mockEmbed := &mockEmbeddingProviderForRetriever{
		embedding: make([]float32, 768),
	}

	config := &TreeRetrieverConfig{
		Collection:       "test_collection",
		TopKPath:         3,
		TopKLeaf:         20,
		ScoreWeightSim:   0.7,
		ScoreWeightLevel: 0.3,
		MaxLevel:         2,
	}

	retriever := NewTreeRetriever(mockStore, mockEmbed, config)

	// 执行检索
	result, err := retriever.Retrieve(context.Background(), "测试问题", "doc_test")
	if err != nil {
		t.Fatalf("Retrieve() 返回错误: %v", err)
	}

	// 应该返回空结果（而非错误）
	if len(result.Results) != 0 {
		t.Errorf("期望返回空结果，实际返回 %d 个结果", len(result.Results))
	}
}

// TestTreeRetriever_Retrieve_EmbeddingFailure 测试向量生成失败。
func TestTreeRetriever_Retrieve_EmbeddingFailure(t *testing.T) {
	mockStore := &mockVectorStoreForRetriever{}

	mockEmbed := &mockEmbeddingProviderForRetriever{
		err: errors.New("向量服务不可用"),
	}

	config := &TreeRetrieverConfig{
		Collection: "test_collection",
		TopKPath:   3,
		TopKLeaf:   20,
		MaxLevel:   2,
	}

	retriever := NewTreeRetriever(mockStore, mockEmbed, config)

	// 执行检索
	_, err := retriever.Retrieve(context.Background(), "测试问题", "doc_test")
	if err == nil {
		t.Error("期望返回错误（向量生成失败），但没有返回")
	}
}

// TestTreeRetriever_RetrieveLeafNodes 测试叶子层检索。
func TestTreeRetriever_RetrieveLeafNodes(t *testing.T) {
	mockStore := &mockVectorStoreForRetriever{
		searchResults: []*store.SearchResult{
			{
				ID:           "leaf_1",
				DocumentID:   "doc_test",
				DocumentName: "测试文档",
				Content:      "叶子节点1",
			},
			{
				ID:           "leaf_2",
				DocumentID:   "doc_test",
				DocumentName: "测试文档",
				Content:      "叶子节点2",
			},
			{
				ID:           "leaf_3",
				DocumentID:   "doc_test",
				DocumentName: "测试文档",
				Content:      "叶子节点3（应被排除）",
			},
		},
	}

	config := &TreeRetrieverConfig{
		Collection: "test_collection",
		TopKLeaf:   20,
	}

	retriever := NewTreeRetriever(mockStore, nil, config)

	queryEmbedding := make([]float32, 768)

	// 排除 leaf_3
	excludeIDs := map[string]bool{
		"leaf_3": true,
	}

	// 执行叶子层检索
	leafNodes, err := retriever.retrieveLeafNodes(context.Background(), queryEmbedding, "doc_test", excludeIDs)
	if err != nil {
		t.Fatalf("retrieveLeafNodes() 返回错误: %v", err)
	}

	// 验证结果
	if len(leafNodes) != 2 {
		t.Errorf("叶子节点数 = %d, 期望 = 2", len(leafNodes))
	}

	// 验证 leaf_3 被排除
	for _, node := range leafNodes {
		if node.ID == "leaf_3" {
			t.Error("leaf_3 应该被排除，但仍在结果中")
		}
	}

	// 验证节点属性
	for i, node := range leafNodes {
		if node.Level != 0 {
			t.Errorf("节点[%d].Level = %d, 期望 = 0", i, node.Level)
		}
		if node.NodeType != 0 {
			t.Errorf("节点[%d].NodeType = %d, 期望 = 0", i, node.NodeType)
		}
	}
}

// TestTreeRetriever_RankAndMerge 测试综合排序和去重。
func TestTreeRetriever_RankAndMerge(t *testing.T) {
	config := &TreeRetrieverConfig{
		ScoreWeightSim:   0.7,
		ScoreWeightLevel: 0.3,
	}

	retriever := NewTreeRetriever(nil, nil, config)

	// 创建测试节点（不同层级、不同相似度）
	pathNodes := []*TreeNode{
		{
			ID:        "node_1",
			Content:   "高层节点",
			Embedding: []float32{1.0, 0.0, 0.0}, // 高相似度
			Level:     2,                        // 根节点
		},
		{
			ID:        "node_2",
			Content:   "中层节点",
			Embedding: []float32{0.5, 0.5, 0.0}, // 中等相似度
			Level:     1,                        // 中间节点
		},
	}

	leafNodes := []*TreeNode{
		{
			ID:        "node_3",
			Content:   "叶子节点",
			Embedding: []float32{0.8, 0.2, 0.0}, // 较高相似度
			Level:     0,                        // 叶子节点
		},
		{
			ID:        "node_1", // 重复节点（应被去重）
			Content:   "高层节点（重复）",
			Embedding: []float32{1.0, 0.0, 0.0},
			Level:     2,
		},
	}

	queryEmbedding := []float32{1.0, 0.0, 0.0}

	// 执行排序和合并
	results := retriever.rankAndMerge(pathNodes, leafNodes, queryEmbedding)

	// 验证去重（应该只有3个节点）
	if len(results) != 3 {
		t.Errorf("结果数 = %d, 期望 = 3（去重后）", len(results))
	}

	// 验证排序（node_1 应该排在最前面：高相似度 + 高层级权重）
	if results[0].ID != "node_1" {
		t.Errorf("排序第一的节点 ID = %q, 期望 = %q", results[0].ID, "node_1")
	}

	// 验证评分字段存在
	for i, result := range results {
		if result.Score == 0 {
			t.Errorf("结果[%d].Score = 0, 应该有评分", i)
		}
		t.Logf("结果[%d]: ID=%s, Score=%.4f", i, result.ID, result.Score)
	}
}

// TestTreeRetriever_CalculateLevelWeight 测试层级权重计算。
func TestTreeRetriever_CalculateLevelWeight(t *testing.T) {
	config := &TreeRetrieverConfig{}
	retriever := NewTreeRetriever(nil, nil, config)

	tests := []struct {
		level          int
		expectedWeight float64
		delta          float64
	}{
		{
			level:          0,
			expectedWeight: 0.3,
			delta:          0.001,
		},
		{
			level:          1,
			expectedWeight: 0.6,
			delta:          0.001,
		},
		{
			level:          2,
			expectedWeight: 0.9,
			delta:          0.001,
		},
		{
			level:          3,
			expectedWeight: 1.0, // 超过最大值，应被限制为 1.0
			delta:          0.001,
		},
		{
			level:          5,
			expectedWeight: 1.0, // 超过最大值，应被限制为 1.0
			delta:          0.001,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Level_%d", tt.level), func(t *testing.T) {
			weight := retriever.calculateLevelWeight(tt.level)
			if abs(weight-tt.expectedWeight) > tt.delta {
				t.Errorf("calculateLevelWeight(%d) = %.3f, 期望 = %.3f (±%.3f)",
					tt.level, weight, tt.expectedWeight, tt.delta)
			}
		})
	}
}

// TestTreeRetriever_DeduplicateNodes 测试去重逻辑。
func TestTreeRetriever_DeduplicateNodes(t *testing.T) {
	config := &TreeRetrieverConfig{}
	retriever := NewTreeRetriever(nil, nil, config)

	nodes := []*TreeNode{
		{ID: "node_1", Content: "内容1"},
		{ID: "node_2", Content: "内容2"},
		{ID: "node_1", Content: "内容1（重复）"},
		{ID: "node_3", Content: "内容3"},
		{ID: "node_2", Content: "内容2（重复）"},
	}

	uniqueNodes := retriever.deduplicateNodes(nodes)

	if len(uniqueNodes) != 3 {
		t.Errorf("去重后节点数 = %d, 期望 = 3", len(uniqueNodes))
	}

	// 验证去重后的节点 ID
	ids := make(map[string]bool)
	for _, node := range uniqueNodes {
		if ids[node.ID] {
			t.Errorf("去重后仍有重复节点: %s", node.ID)
		}
		ids[node.ID] = true
	}
}

// 辅助函数
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
