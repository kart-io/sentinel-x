package biz

import (
	"context"
	"errors"
	"testing"

	"github.com/kart-io/sentinel-x/internal/rag/store"
)

// === Mock 实现 ===

// mockVectorStoreForPath 模拟 VectorStore 用于 PathFinder 测试。
type mockVectorStoreForPath struct {
	// 模拟数据：根节点、子节点映射
	rootNodes         []*store.SearchResult
	childrenMap       map[string][]*store.SearchResult // parent_id -> children
	searchError       error
	shouldReturnEmpty bool
}

func (m *mockVectorStoreForPath) SearchWithFilter(ctx context.Context, collection string, embedding []float32, expr string, topK int) ([]*store.SearchResult, error) {
	if m.searchError != nil {
		return nil, m.searchError
	}

	if m.shouldReturnEmpty {
		return []*store.SearchResult{}, nil
	}

	// 解析过滤表达式，判断是查询根节点还是子节点
	// 简化实现：根据 expr 包含的关键字判断
	if contains(expr, "node_type == 2") {
		// 查询根节点
		return m.rootNodes, nil
	} else if contains(expr, "parent_id ==") {
		// 查询子节点：从 expr 中提取 parent_id
		parentID := extractParentID(expr)
		if children, ok := m.childrenMap[parentID]; ok {
			return children, nil
		}
		return []*store.SearchResult{}, nil
	}

	return []*store.SearchResult{}, nil
}

func (m *mockVectorStoreForPath) Search(ctx context.Context, collection string, embedding []float32, topK int) ([]*store.SearchResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockVectorStoreForPath) Insert(ctx context.Context, collection string, chunks []*store.Chunk) ([]string, error) {
	return nil, errors.New("not implemented")
}

func (m *mockVectorStoreForPath) CreateCollection(ctx context.Context, config *store.CollectionConfig) error {
	return errors.New("not implemented")
}

func (m *mockVectorStoreForPath) GetStats(ctx context.Context, collection string) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *mockVectorStoreForPath) Close(ctx context.Context) error {
	return nil
}

// 接口实现验证
var _ store.VectorStore = (*mockVectorStoreForPath)(nil)

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func extractParentID(expr string) string {
	// 简化实现：从 "parent_id == 'xxx'" 中提取 xxx
	// 这里使用简单的字符串处理
	start := -1
	end := -1
	inQuote := false
	for i, ch := range expr {
		if ch == '\'' {
			if !inQuote {
				start = i + 1
				inQuote = true
			} else {
				end = i
				break
			}
		}
	}
	if start >= 0 && end > start {
		return expr[start:end]
	}
	return ""
}

// === 测试用例 ===

// TestPathFinder_FindPaths_Success 测试成功的路径查找。
func TestPathFinder_FindPaths_Success(t *testing.T) {
	// 准备测试数据：三层树结构
	// Level 2: root_1
	// Level 1: child_1_1, child_1_2
	// Level 0: leaf_1_1, leaf_1_2

	mockStore := &mockVectorStoreForPath{
		rootNodes: []*store.SearchResult{
			{
				ID:           "root_1",
				DocumentID:   "doc_test",
				DocumentName: "测试文档",
				Section:      "section_1",
				Content:      "根节点内容",
				Score:        0.9,
			},
		},
		childrenMap: map[string][]*store.SearchResult{
			"root_1": {
				{
					ID:           "child_1_1",
					DocumentID:   "doc_test",
					DocumentName: "测试文档",
					Section:      "section_1",
					Content:      "子节点1内容",
					Score:        0.8,
				},
				{
					ID:           "child_1_2",
					DocumentID:   "doc_test",
					DocumentName: "测试文档",
					Section:      "section_1",
					Content:      "子节点2内容",
					Score:        0.7,
				},
			},
			"child_1_1": {
				{
					ID:           "leaf_1_1",
					DocumentID:   "doc_test",
					DocumentName: "测试文档",
					Section:      "section_1",
					Content:      "叶子节点1内容",
					Score:        0.85,
				},
			},
			"child_1_2": {
				{
					ID:           "leaf_1_2",
					DocumentID:   "doc_test",
					DocumentName: "测试文档",
					Section:      "section_1",
					Content:      "叶子节点2内容",
					Score:        0.75,
				},
			},
		},
	}

	config := &PathFinderConfig{
		Collection: "test_collection",
		TopK:       2,
		MaxLevel:   2,
	}

	pathFinder := NewPathFinder(mockStore, config)

	// 创建查询向量（全1向量，简化测试）
	queryEmbedding := make([]float32, 768)
	for i := range queryEmbedding {
		queryEmbedding[i] = 1.0
	}

	// 执行路径查找
	pathNodes, err := pathFinder.FindPaths(context.Background(), queryEmbedding, "doc_test")
	if err != nil {
		t.Fatalf("FindPaths() 返回错误: %v", err)
	}

	// 验证结果
	if len(pathNodes) == 0 {
		t.Error("FindPaths() 返回空结果")
	}

	// 应该包含根节点
	hasRoot := false
	for _, node := range pathNodes {
		if node.ID == "root_1" {
			hasRoot = true
			break
		}
	}
	if !hasRoot {
		t.Error("路径中未包含根节点")
	}

	t.Logf("找到 %d 个路径节点", len(pathNodes))
}

// TestPathFinder_FindPaths_NoRootNodes 测试没有根节点的情况。
func TestPathFinder_FindPaths_NoRootNodes(t *testing.T) {
	mockStore := &mockVectorStoreForPath{
		shouldReturnEmpty: true,
	}

	config := &PathFinderConfig{
		Collection: "test_collection",
		TopK:       2,
		MaxLevel:   2,
	}

	pathFinder := NewPathFinder(mockStore, config)

	queryEmbedding := make([]float32, 768)

	pathNodes, err := pathFinder.FindPaths(context.Background(), queryEmbedding, "doc_test")
	if err != nil {
		t.Fatalf("FindPaths() 返回错误: %v", err)
	}

	// 应该返回空数组（而非错误）
	if len(pathNodes) != 0 {
		t.Errorf("期望返回空数组，实际返回 %d 个节点", len(pathNodes))
	}
}

// TestPathFinder_FindPaths_SearchError 测试查询失败的情况。
func TestPathFinder_FindPaths_SearchError(t *testing.T) {
	mockStore := &mockVectorStoreForPath{
		searchError: errors.New("向量存储不可用"),
	}

	config := &PathFinderConfig{
		Collection: "test_collection",
		TopK:       2,
		MaxLevel:   2,
	}

	pathFinder := NewPathFinder(mockStore, config)

	queryEmbedding := make([]float32, 768)

	_, err := pathFinder.FindPaths(context.Background(), queryEmbedding, "doc_test")
	if err == nil {
		t.Error("期望返回错误（查询失败），但没有返回")
	}
}

// TestPathFinder_FindRootNodes 测试查找根节点。
func TestPathFinder_FindRootNodes(t *testing.T) {
	mockStore := &mockVectorStoreForPath{
		rootNodes: []*store.SearchResult{
			{
				ID:           "root_1",
				DocumentID:   "doc_test",
				DocumentName: "测试文档",
				Content:      "根节点1",
			},
			{
				ID:           "root_2",
				DocumentID:   "doc_test",
				DocumentName: "测试文档",
				Content:      "根节点2",
			},
		},
	}

	config := &PathFinderConfig{
		Collection: "test_collection",
		MaxLevel:   2,
	}

	pathFinder := NewPathFinder(mockStore, config)

	rootNodes, err := pathFinder.findRootNodes(context.Background(), "doc_test")
	if err != nil {
		t.Fatalf("findRootNodes() 返回错误: %v", err)
	}

	if len(rootNodes) != 2 {
		t.Errorf("根节点数 = %d, 期望 = 2", len(rootNodes))
	}

	// 验证节点属性
	for i, node := range rootNodes {
		if node.NodeType != 2 {
			t.Errorf("节点[%d].NodeType = %d, 期望 = 2", i, node.NodeType)
		}
		if node.Level != 2 {
			t.Errorf("节点[%d].Level = %d, 期望 = 2", i, node.Level)
		}
	}
}

// TestPathFinder_SelectTopK 测试选择 topK 节点。
func TestPathFinder_SelectTopK(t *testing.T) {
	config := &PathFinderConfig{
		Collection: "test_collection",
	}

	pathFinder := NewPathFinder(nil, config)

	// 创建测试节点（带 embedding）
	nodes := []*TreeNode{
		{
			ID:        "node_1",
			Content:   "节点1",
			Embedding: []float32{1.0, 0.0, 0.0}, // 与查询向量相似度最高
		},
		{
			ID:        "node_2",
			Content:   "节点2",
			Embedding: []float32{0.0, 1.0, 0.0},
		},
		{
			ID:        "node_3",
			Content:   "节点3",
			Embedding: []float32{0.8, 0.2, 0.0}, // 与查询向量相似度次高
		},
	}

	queryEmbedding := []float32{1.0, 0.0, 0.0}

	// 选择 top 2
	topNodes := pathFinder.selectTopK(nodes, queryEmbedding, 2)

	if len(topNodes) != 2 {
		t.Fatalf("topNodes 长度 = %d, 期望 = 2", len(topNodes))
	}

	// 验证排序结果（应该是 node_1 和 node_3）
	if topNodes[0].ID != "node_1" {
		t.Errorf("topNodes[0].ID = %q, 期望 = %q", topNodes[0].ID, "node_1")
	}

	if topNodes[1].ID != "node_3" {
		t.Errorf("topNodes[1].ID = %q, 期望 = %q", topNodes[1].ID, "node_3")
	}
}

// TestPathFinder_SelectTopK_LessThanK 测试节点数少于 k 的情况。
func TestPathFinder_SelectTopK_LessThanK(t *testing.T) {
	config := &PathFinderConfig{
		Collection: "test_collection",
	}

	pathFinder := NewPathFinder(nil, config)

	nodes := []*TreeNode{
		{ID: "node_1", Embedding: []float32{1.0, 0.0, 0.0}},
		{ID: "node_2", Embedding: []float32{0.0, 1.0, 0.0}},
	}

	queryEmbedding := []float32{1.0, 0.0, 0.0}

	// 选择 top 5，但只有 2 个节点
	topNodes := pathFinder.selectTopK(nodes, queryEmbedding, 5)

	if len(topNodes) != 2 {
		t.Errorf("topNodes 长度 = %d, 期望 = 2", len(topNodes))
	}
}

// TestPathFinder_DeduplicateNodes 测试去重逻辑。
func TestPathFinder_DeduplicateNodes(t *testing.T) {
	config := &PathFinderConfig{
		Collection: "test_collection",
	}

	pathFinder := NewPathFinder(nil, config)

	nodes := []*TreeNode{
		{ID: "node_1", Content: "内容1"},
		{ID: "node_2", Content: "内容2"},
		{ID: "node_1", Content: "内容1（重复）"},
		{ID: "node_3", Content: "内容3"},
		{ID: "node_2", Content: "内容2（重复）"},
	}

	uniqueNodes := pathFinder.deduplicateNodes(nodes)

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

// TestPathFinder_TraverseDown_SingleLevel 测试单层递归。
func TestPathFinder_TraverseDown_SingleLevel(t *testing.T) {
	mockStore := &mockVectorStoreForPath{
		childrenMap: map[string][]*store.SearchResult{
			"parent_1": {
				{
					ID:      "leaf_1",
					Content: "叶子节点",
				},
			},
		},
	}

	config := &PathFinderConfig{
		Collection: "test_collection",
		TopK:       1,
		MaxLevel:   1,
	}

	pathFinder := NewPathFinder(mockStore, config)

	// 创建父节点（Level 1）
	parentNode := &TreeNode{
		ID:       "parent_1",
		Content:  "父节点",
		Level:    1,
		NodeType: 1,
	}

	queryEmbedding := make([]float32, 768)

	// 执行递归向下查找
	pathNodes, err := pathFinder.traverseDown(context.Background(), parentNode, queryEmbedding)
	if err != nil {
		t.Fatalf("traverseDown() 返回错误: %v", err)
	}

	// 应该包含父节点和叶子节点
	if len(pathNodes) < 1 {
		t.Errorf("路径节点数 = %d, 期望 >= 1", len(pathNodes))
	}

	// 验证包含父节点
	if pathNodes[0].ID != "parent_1" {
		t.Errorf("路径首节点 ID = %q, 期望 = %q", pathNodes[0].ID, "parent_1")
	}
}

// TestPathFinder_TraverseDown_LeafNode 测试叶子节点（停止条件）。
func TestPathFinder_TraverseDown_LeafNode(t *testing.T) {
	config := &PathFinderConfig{
		Collection: "test_collection",
		TopK:       1,
		MaxLevel:   2,
	}

	pathFinder := NewPathFinder(nil, config)

	// 创建叶子节点（Level 0）
	leafNode := &TreeNode{
		ID:       "leaf_1",
		Content:  "叶子节点",
		Level:    0, // 叶子节点
		NodeType: 0,
	}

	queryEmbedding := make([]float32, 768)

	// 执行递归向下查找
	pathNodes, err := pathFinder.traverseDown(context.Background(), leafNode, queryEmbedding)
	if err != nil {
		t.Fatalf("traverseDown() 返回错误: %v", err)
	}

	// 叶子节点应该立即返回，路径长度为 1
	if len(pathNodes) != 1 {
		t.Errorf("路径节点数 = %d, 期望 = 1", len(pathNodes))
	}

	if pathNodes[0].ID != "leaf_1" {
		t.Errorf("路径节点 ID = %q, 期望 = %q", pathNodes[0].ID, "leaf_1")
	}
}
