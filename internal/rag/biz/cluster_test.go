package biz

import (
	"math"
	"testing"
)

// TestKMeansClusterer_CosineSimilarity 测试余弦相似度计算。
func TestKMeansClusterer_CosineSimilarity(t *testing.T) {
	clusterer := NewKMeansClusterer(nil)

	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
		delta    float32 // 允许的误差范围
	}{
		{
			name:     "完全相同的向量",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 1.0,
			delta:    0.0001,
		},
		{
			name:     "完全相反的向量",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{-1.0, 0.0, 0.0},
			expected: -1.0,
			delta:    0.0001,
		},
		{
			name:     "正交向量",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{0.0, 1.0, 0.0},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "45度角向量",
			a:        []float32{1.0, 0.0},
			b:        []float32{1.0, 1.0},
			expected: 0.7071, // cos(45°) ≈ 0.7071
			delta:    0.001,
		},
		{
			name:     "零向量处理",
			a:        []float32{0.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 0.0,
			delta:    0.0001,
		},
		{
			name:     "不同维度向量",
			a:        []float32{1.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 0.0, // 维度不匹配返回0
			delta:    0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clusterer.cosineSimilarity(tt.a, tt.b)
			if math.Abs(float64(result-tt.expected)) > float64(tt.delta) {
				t.Errorf("cosineSimilarity() = %v, expected %v (±%v)", result, tt.expected, tt.delta)
			}
		})
	}
}

// TestKMeansClusterer_Normalize 测试向量归一化。
func TestKMeansClusterer_Normalize(t *testing.T) {
	clusterer := NewKMeansClusterer(nil)

	tests := []struct {
		name     string
		input    []float32
		expected []float32
		delta    float32
	}{
		{
			name:     "标准向量归一化",
			input:    []float32{3.0, 4.0},
			expected: []float32{0.6, 0.8}, // 长度为5的向量
			delta:    0.0001,
		},
		{
			name:     "已归一化向量",
			input:    []float32{1.0, 0.0, 0.0},
			expected: []float32{1.0, 0.0, 0.0},
			delta:    0.0001,
		},
		{
			name:     "零向量处理",
			input:    []float32{0.0, 0.0, 0.0},
			expected: []float32{0.0, 0.0, 0.0}, // 零向量保持不变
			delta:    0.0001,
		},
		{
			name:     "三维向量归一化",
			input:    []float32{1.0, 2.0, 2.0},
			expected: []float32{0.3333, 0.6667, 0.6667}, // 长度为3
			delta:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clusterer.normalize(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("normalize() 返回长度 = %d, 期望 %d", len(result), len(tt.expected))
			}

			for i := 0; i < len(result); i++ {
				if math.Abs(float64(result[i]-tt.expected[i])) > float64(tt.delta) {
					t.Errorf("normalize()[%d] = %v, expected %v (±%v)", i, result[i], tt.expected[i], tt.delta)
				}
			}

			// 验证归一化后的向量长度为1（零向量除外）
			if !isZeroVector(tt.input) {
				var norm float32
				for _, v := range result {
					norm += v * v
				}
				norm = float32(math.Sqrt(float64(norm)))
				if math.Abs(float64(norm-1.0)) > 0.001 {
					t.Errorf("normalize() 向量长度 = %v, 期望 1.0", norm)
				}
			}
		})
	}
}

// TestKMeansClusterer_HasConverged 测试分配收敛检测。
func TestKMeansClusterer_HasConverged(t *testing.T) {
	clusterer := NewKMeansClusterer(nil)

	tests := []struct {
		name           string
		oldAssignments []int
		newAssignments []int
		expected       bool
	}{
		{
			name:           "完全相同的分配",
			oldAssignments: []int{0, 1, 0, 1, 2},
			newAssignments: []int{0, 1, 0, 1, 2},
			expected:       true,
		},
		{
			name:           "部分不同的分配",
			oldAssignments: []int{0, 1, 0, 1, 2},
			newAssignments: []int{0, 1, 0, 2, 2},
			expected:       false,
		},
		{
			name:           "长度不匹配",
			oldAssignments: []int{0, 1, 0},
			newAssignments: []int{0, 1},
			expected:       false,
		},
		{
			name:           "空分配",
			oldAssignments: []int{},
			newAssignments: []int{},
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clusterer.hasConverged(tt.oldAssignments, tt.newAssignments)
			if result != tt.expected {
				t.Errorf("hasConverged() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestKMeansClusterer_CentersConverged 测试中心收敛检测。
func TestKMeansClusterer_CentersConverged(t *testing.T) {
	config := &KMeansClustererConfig{
		MaxIterations:        10,
		ConvergenceThreshold: 0.01, // 1% 阈值
	}
	clusterer := NewKMeansClusterer(config)

	tests := []struct {
		name       string
		oldCenters [][]float32
		newCenters [][]float32
		expected   bool
	}{
		{
			name: "完全相同的中心",
			oldCenters: [][]float32{
				{1.0, 0.0, 0.0},
				{0.0, 1.0, 0.0},
			},
			newCenters: [][]float32{
				{1.0, 0.0, 0.0},
				{0.0, 1.0, 0.0},
			},
			expected: true,
		},
		{
			name: "微小差异（收敛）",
			oldCenters: [][]float32{
				{1.0, 0.0, 0.0},
			},
			newCenters: [][]float32{
				{0.995, 0.0, 0.0}, // 相似度 > 0.99
			},
			expected: true,
		},
		{
			name: "显著差异（未收敛）",
			oldCenters: [][]float32{
				{1.0, 0.0, 0.0},
			},
			newCenters: [][]float32{
				{0.8, 0.6, 0.0}, // 方向变化，相似度约0.8 < 0.99
			},
			expected: false,
		},
		{
			name: "长度不匹配",
			oldCenters: [][]float32{
				{1.0, 0.0, 0.0},
			},
			newCenters: [][]float32{
				{1.0, 0.0, 0.0},
				{0.0, 1.0, 0.0},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clusterer.centersConverged(tt.oldCenters, tt.newCenters)
			if result != tt.expected {
				t.Errorf("centersConverged() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestKMeansClusterer_Cluster_BoundaryConditions 测试边界条件。
func TestKMeansClusterer_Cluster_BoundaryConditions(t *testing.T) {
	clusterer := NewKMeansClusterer(nil)

	t.Run("空节点列表", func(t *testing.T) {
		_, err := clusterer.Cluster(nil, 3)
		if err == nil {
			t.Error("期望返回错误，但没有返回")
		}
	})

	t.Run("k <= 0", func(t *testing.T) {
		nodes := createTestNodes(5)
		_, err := clusterer.Cluster(nodes, 0)
		if err == nil {
			t.Error("期望返回错误，但没有返回")
		}
	})

	t.Run("节点数 <= k (每个节点单独成簇)", func(t *testing.T) {
		nodes := createTestNodes(3)
		clusters, err := clusterer.Cluster(nodes, 5)
		if err != nil {
			t.Fatalf("Cluster() 返回错误: %v", err)
		}

		if len(clusters) != 3 {
			t.Errorf("期望簇数 = 3, 实际 = %d", len(clusters))
		}

		// 验证每个簇只有一个节点
		for i, cluster := range clusters {
			if len(cluster) != 1 {
				t.Errorf("簇 %d 节点数 = %d, 期望 = 1", i, len(cluster))
			}
		}
	})

	t.Run("节点数 <= 5 (所有节点在一个簇中)", func(t *testing.T) {
		nodes := createTestNodes(5)
		clusters, err := clusterer.Cluster(nodes, 3)
		if err != nil {
			t.Fatalf("Cluster() 返回错误: %v", err)
		}

		if len(clusters) != 1 {
			t.Errorf("期望簇数 = 1, 实际 = %d", len(clusters))
		}

		if len(clusters[0]) != 5 {
			t.Errorf("簇节点数 = %d, 期望 = 5", len(clusters[0]))
		}
	})
}

// TestKMeansClusterer_Cluster_SimpleScenario 测试简单聚类场景。
func TestKMeansClusterer_Cluster_SimpleScenario(t *testing.T) {
	clusterer := NewKMeansClusterer(&KMeansClustererConfig{
		MaxIterations:        10,
		ConvergenceThreshold: 0.001,
	})

	// 创建两组明显分离的节点
	nodes := []*TreeNode{
		// 簇1: 靠近 (1, 0, 0)
		{ID: "1", Embedding: normalize([]float32{1.0, 0.1, 0.1})},
		{ID: "2", Embedding: normalize([]float32{0.9, 0.0, 0.0})},
		{ID: "3", Embedding: normalize([]float32{1.0, 0.2, 0.0})},
		// 簇2: 靠近 (0, 1, 0)
		{ID: "4", Embedding: normalize([]float32{0.0, 1.0, 0.1})},
		{ID: "5", Embedding: normalize([]float32{0.1, 0.9, 0.0})},
		{ID: "6", Embedding: normalize([]float32{0.0, 1.0, 0.2})},
		// 簇3: 靠近 (0, 0, 1)
		{ID: "7", Embedding: normalize([]float32{0.0, 0.1, 1.0})},
		{ID: "8", Embedding: normalize([]float32{0.1, 0.0, 0.9})},
		{ID: "9", Embedding: normalize([]float32{0.0, 0.2, 1.0})},
	}

	clusters, err := clusterer.Cluster(nodes, 3)
	if err != nil {
		t.Fatalf("Cluster() 返回错误: %v", err)
	}

	// 验证簇数
	if len(clusters) != 3 {
		t.Errorf("期望簇数 = 3, 实际 = %d", len(clusters))
	}

	// 验证每个簇的节点数
	totalNodes := 0
	for i, cluster := range clusters {
		totalNodes += len(cluster)
		t.Logf("簇 %d 节点数: %d", i, len(cluster))
	}

	if totalNodes != 9 {
		t.Errorf("总节点数 = %d, 期望 = 9", totalNodes)
	}

	// 验证每个簇内的节点相似度较高
	for i, cluster := range clusters {
		if len(cluster) < 2 {
			continue
		}

		// 计算簇内平均相似度
		avgSimilarity := calculateClusterSimilarity(cluster, clusterer)
		t.Logf("簇 %d 平均相似度: %.4f", i, avgSimilarity)

		// 簇内相似度应该较高（> 0.8）
		if avgSimilarity < 0.8 {
			t.Errorf("簇 %d 平均相似度 = %.4f, 期望 > 0.8", i, avgSimilarity)
		}
	}
}

// TestKMeansClusterer_Cluster_FilterEmptyClusters 测试空簇过滤。
func TestKMeansClusterer_Cluster_FilterEmptyClusters(t *testing.T) {
	clusterer := NewKMeansClusterer(nil)

	// 创建10个节点，聚类为3个簇
	nodes := createTestNodes(10)
	clusters, err := clusterer.Cluster(nodes, 3)
	if err != nil {
		t.Fatalf("Cluster() 返回错误: %v", err)
	}

	// 验证没有空簇
	for i, cluster := range clusters {
		if len(cluster) == 0 {
			t.Errorf("簇 %d 是空簇", i)
		}
	}

	// 验证所有节点都被分配
	totalNodes := 0
	for _, cluster := range clusters {
		totalNodes += len(cluster)
	}

	if totalNodes != 10 {
		t.Errorf("总节点数 = %d, 期望 = 10", totalNodes)
	}
}

// === 辅助函数 ===

// createTestNodes 创建测试节点（随机向量）。
func createTestNodes(count int) []*TreeNode {
	nodes := make([]*TreeNode, count)
	for i := 0; i < count; i++ {
		// 创建简单的测试向量
		embedding := make([]float32, 3)
		embedding[i%3] = 1.0 // 轮流在不同维度上设置1.0
		nodes[i] = &TreeNode{
			ID:        string(rune('A' + i)),
			Content:   "测试内容",
			Embedding: embedding,
			Level:     0,
		}
	}
	return nodes
}

// normalize 归一化向量（辅助函数）。
func normalize(vec []float32) []float32 {
	var norm float32
	for _, v := range vec {
		norm += v * v
	}
	if norm == 0 {
		return vec
	}
	norm = float32(math.Sqrt(float64(norm)))
	normalized := make([]float32, len(vec))
	for i, v := range vec {
		normalized[i] = v / norm
	}
	return normalized
}

// isZeroVector 判断是否为零向量。
func isZeroVector(vec []float32) bool {
	for _, v := range vec {
		if v != 0 {
			return false
		}
	}
	return true
}

// calculateClusterSimilarity 计算簇内平均相似度。
func calculateClusterSimilarity(cluster []*TreeNode, clusterer *KMeansClusterer) float32 {
	if len(cluster) < 2 {
		return 1.0
	}

	totalSim := float32(0.0)
	count := 0

	for i := 0; i < len(cluster); i++ {
		for j := i + 1; j < len(cluster); j++ {
			sim := clusterer.cosineSimilarity(cluster[i].Embedding, cluster[j].Embedding)
			totalSim += sim
			count++
		}
	}

	if count == 0 {
		return 1.0
	}

	return totalSim / float32(count)
}
