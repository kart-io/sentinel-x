package biz

import (
	"fmt"
	"math"
	"math/rand"
)

// TreeNode 表示树中的一个节点。
type TreeNode struct {
	// ID 节点唯一标识。
	ID string
	// Content 节点内容（叶子节点）或摘要（父节点）。
	Content string
	// Embedding 节点的向量表示。
	Embedding []float32
	// Level 节点在树中的层级（0=叶子节点）。
	Level int
	// ParentID 父节点 ID（根节点为空字符串）。
	ParentID string
	// NodeType 节点类型（0=叶子，1=中间节点，2=根节点）。
	NodeType int
	// DocumentID 所属文档 ID。
	DocumentID string
	// DocumentName 文档名称。
	DocumentName string
	// Section 所属章节。
	Section string
}

// KMeansClustererConfig KMeans 聚类器配置。
type KMeansClustererConfig struct {
	// MaxIterations 最大迭代次数。
	MaxIterations int
	// ConvergenceThreshold 收敛阈值。
	ConvergenceThreshold float32
}

// KMeansClusterer 实现基于向量相似度的 KMeans 聚类算法。
type KMeansClusterer struct {
	config *KMeansClustererConfig
}

// NewKMeansClusterer 创建 KMeans 聚类器实例。
func NewKMeansClusterer(config *KMeansClustererConfig) *KMeansClusterer {
	if config == nil {
		config = &KMeansClustererConfig{
			MaxIterations:        10,
			ConvergenceThreshold: 0.001,
		}
	}
	return &KMeansClusterer{config: config}
}

// Cluster 将节点聚类为 k 个簇。
// 使用 KMeans 算法基于向量的余弦相似度进行聚类。
func (c *KMeansClusterer) Cluster(nodes []*TreeNode, k int) ([][]*TreeNode, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("节点列表为空")
	}

	if k <= 0 {
		return nil, fmt.Errorf("簇数量必须大于 0")
	}

	// 边界条件1：如果节点数 <= 簇数，每个节点单独成簇
	if len(nodes) <= k {
		clusters := make([][]*TreeNode, len(nodes))
		for i, node := range nodes {
			clusters[i] = []*TreeNode{node}
		}
		return clusters, nil
	}

	// 边界条件2：如果节点数 <= 5，不再聚类，所有节点放在一个簇中
	if len(nodes) <= 5 {
		return [][]*TreeNode{nodes}, nil
	}

	// 1. 初始化聚类中心
	centers := c.initializeCenters(nodes, k)

	var assignments []int
	// 2. 迭代直到收敛或达到最大迭代次数
	for iter := 0; iter < c.config.MaxIterations; iter++ {
		// 分配节点到最近的中心
		newAssignments := c.assignClusters(nodes, centers)

		// 检查是否收敛（分配结果不再变化）
		if iter > 0 && c.hasConverged(assignments, newAssignments) {
			assignments = newAssignments
			break
		}

		assignments = newAssignments

		// 更新聚类中心
		newCenters := c.updateCenters(nodes, assignments, k)

		// 检查中心是否变化
		if c.centersConverged(centers, newCenters) {
			break
		}

		centers = newCenters
	}

	// 3. 构建最终的聚类结果
	clusters := make([][]*TreeNode, k)
	for i, assignment := range assignments {
		clusters[assignment] = append(clusters[assignment], nodes[i])
	}

	// 4. 过滤空簇
	nonEmptyClusters := make([][]*TreeNode, 0, k)
	for _, cluster := range clusters {
		if len(cluster) > 0 {
			nonEmptyClusters = append(nonEmptyClusters, cluster)
		}
	}

	return nonEmptyClusters, nil
}

// initializeCenters 随机初始化 k 个聚类中心。
// 使用 k-means++ 初始化策略以获得更好的初始中心。
func (c *KMeansClusterer) initializeCenters(nodes []*TreeNode, k int) [][]float32 {
	centers := make([][]float32, k)

	// 随机选择第一个中心
	//nolint:gosec // G404: 用于文档聚类算法，非安全敏感场景，可以使用 math/rand
	firstIdx := rand.Intn(len(nodes))
	centers[0] = nodes[firstIdx].Embedding

	// 使用 k-means++ 策略选择后续中心
	for i := 1; i < k; i++ {
		// 计算每个节点到最近中心的距离
		distances := make([]float32, len(nodes))
		totalDist := float32(0.0)

		for j, node := range nodes {
			minDist := float32(2.0) // 余弦距离最大为 2
			for _, center := range centers[:i] {
				dist := 1.0 - c.cosineSimilarity(node.Embedding, center)
				if dist < minDist {
					minDist = dist
				}
			}
			distances[j] = minDist * minDist // 距离的平方
			totalDist += distances[j]
		}

		// 按概率选择下一个中心（距离越远概率越大）
		//nolint:gosec // G404: 用于文档聚类算法，非安全敏感场景，可以使用 math/rand
		r := rand.Float32() * totalDist
		cumulative := float32(0.0)
		for j, dist := range distances {
			cumulative += dist
			if cumulative >= r {
				centers[i] = nodes[j].Embedding
				break
			}
		}
	}

	return centers
}

// assignClusters 将每个节点分配到最近的聚类中心。
func (c *KMeansClusterer) assignClusters(nodes []*TreeNode, centers [][]float32) []int {
	assignments := make([]int, len(nodes))

	for i, node := range nodes {
		maxSim := float32(-1.0)
		bestCluster := 0

		for j, center := range centers {
			sim := c.cosineSimilarity(node.Embedding, center)
			if sim > maxSim {
				maxSim = sim
				bestCluster = j
			}
		}

		assignments[i] = bestCluster
	}

	return assignments
}

// updateCenters 重新计算每个簇的中心（向量平均）。
func (c *KMeansClusterer) updateCenters(nodes []*TreeNode, assignments []int, k int) [][]float32 {
	// 统计每个簇的节点数
	clusterSizes := make([]int, k)
	for _, assignment := range assignments {
		clusterSizes[assignment]++
	}

	// 计算向量维度（假设所有向量维度相同）
	dim := len(nodes[0].Embedding)

	// 初始化新中心
	newCenters := make([][]float32, k)
	for i := 0; i < k; i++ {
		newCenters[i] = make([]float32, dim)
	}

	// 累加每个簇的向量
	for i, node := range nodes {
		clusterIdx := assignments[i]
		for d := 0; d < dim; d++ {
			newCenters[clusterIdx][d] += node.Embedding[d]
		}
	}

	// 计算平均（归一化）
	for i := 0; i < k; i++ {
		if clusterSizes[i] > 0 {
			for d := 0; d < dim; d++ {
				newCenters[i][d] /= float32(clusterSizes[i])
			}
			// 归一化向量（单位向量）
			newCenters[i] = c.normalize(newCenters[i])
		}
	}

	return newCenters
}

// cosineSimilarity 计算两个向量的余弦相似度。
// 返回值范围：[-1, 1]，值越大表示越相似。
func (c *KMeansClusterer) cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// normalize 归一化向量为单位向量。
func (c *KMeansClusterer) normalize(vec []float32) []float32 {
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

// hasConverged 检查分配结果是否收敛（不再变化）。
func (c *KMeansClusterer) hasConverged(oldAssignments, newAssignments []int) bool {
	if len(oldAssignments) != len(newAssignments) {
		return false
	}

	for i := 0; i < len(oldAssignments); i++ {
		if oldAssignments[i] != newAssignments[i] {
			return false
		}
	}

	return true
}

// centersConverged 检查聚类中心是否收敛。
func (c *KMeansClusterer) centersConverged(oldCenters, newCenters [][]float32) bool {
	if len(oldCenters) != len(newCenters) {
		return false
	}

	for i := 0; i < len(oldCenters); i++ {
		sim := c.cosineSimilarity(oldCenters[i], newCenters[i])
		// 如果相似度 < 1 - threshold，说明还没收敛
		if sim < 1.0-c.config.ConvergenceThreshold {
			return false
		}
	}

	return true
}
