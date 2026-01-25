package biz

import (
	"context"
	"fmt"
	"sort"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/textutil"
	"github.com/kart-io/sentinel-x/internal/rag/store"
)

// PathFinderConfig 路径查找器配置。
type PathFinderConfig struct {
	// Collection 集合名称。
	Collection string
	// TopK 每层返回的节点数量。
	TopK int
	// MaxLevel 树的最大层级。
	MaxLevel int
}

// PathFinder 负责在树形索引中进行自顶向下的路径查找。
// 从根节点开始，递归向下查找与查询最相关的节点路径。
type PathFinder struct {
	store  store.VectorStore
	config *PathFinderConfig
}

// NewPathFinder 创建路径查找器实例。
func NewPathFinder(vectorStore store.VectorStore, config *PathFinderConfig) *PathFinder {
	return &PathFinder{
		store:  vectorStore,
		config: config,
	}
}

// FindPaths 执行自顶向下的路径查找。
// 从根节点开始，递归向下选择最相关的子节点，直到叶子节点。
// 返回路径上的所有节点。
func (pf *PathFinder) FindPaths(ctx context.Context, queryEmbedding []float32, documentID string) ([]*TreeNode, error) {
	logger.Infow("开始路径查找",
		"document_id", documentID,
		"top_k", pf.config.TopK,
	)

	// 检查 context 是否已取消
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context cancelled before path finding: %w", ctx.Err())
	}

	// 1. 查找根节点
	rootNodes, err := pf.findRootNodes(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("查找根节点失败: %w", err)
	}

	if len(rootNodes) == 0 {
		logger.Warnw("未找到根节点，文档可能没有树索引",
			"document_id", documentID,
		)
		return []*TreeNode{}, nil
	}

	logger.Infow("找到根节点",
		"document_id", documentID,
		"root_count", len(rootNodes),
	)

	// 2. 从根节点开始递归向下查找
	var allPathNodes []*TreeNode

	// 选择 topK 个最相关的根节点
	topRootNodes := pf.selectTopK(rootNodes, queryEmbedding, pf.config.TopK)

	for i, rootNode := range topRootNodes {
		logger.Infow("开始从根节点向下查找",
			"root_index", i,
			"root_id", rootNode.ID,
			"level", rootNode.Level,
		)

		// 递归向下查找
		pathNodes, err := pf.traverseDown(ctx, rootNode, queryEmbedding)
		if err != nil {
			logger.Warnw("从根节点向下查找失败，跳过该路径",
				"root_id", rootNode.ID,
				"error", err.Error(),
			)
			continue
		}

		allPathNodes = append(allPathNodes, pathNodes...)

		logger.Infow("路径查找完成",
			"root_id", rootNode.ID,
			"path_nodes_count", len(pathNodes),
		)
	}

	// 去重（可能多条路径经过相同节点）
	uniqueNodes := pf.deduplicateNodes(allPathNodes)

	logger.Infow("路径查找总结",
		"total_paths", len(topRootNodes),
		"total_nodes", len(allPathNodes),
		"unique_nodes", len(uniqueNodes),
	)

	return uniqueNodes, nil
}

// findRootNodes 查找文档的根节点。
// 根节点定义：node_type == 2（根节点类型）。
func (pf *PathFinder) findRootNodes(ctx context.Context, documentID string) ([]*TreeNode, error) {
	// 构建过滤表达式：处理空 documentID
	var filterExpr string
	if documentID == "" {
		// POC 阶段：查询所有文档的根节点
		filterExpr = "node_type == 2"
		logger.Infow("查找根节点（所有文档）", "filter", filterExpr)
	} else {
		// 查询指定文档的根节点
		filterExpr = fmt.Sprintf("document_id == '%s' && node_type == 2", documentID)
		logger.Infow("查找根节点（指定文档）", "document_id", documentID, "filter", filterExpr)
	}

	// 使用零向量进行查询（主要依赖过滤表达式）
	dummyEmbedding := make([]float32, 768) // 假设向量维度为 768

	// 查询根节点
	results, err := pf.store.SearchWithFilter(ctx, pf.config.Collection, dummyEmbedding, filterExpr, 100)
	if err != nil {
		return nil, fmt.Errorf("查询根节点失败: %w", err)
	}

	// 转换为 TreeNode 格式
	nodes := make([]*TreeNode, len(results))
	for i, result := range results {
		nodes[i] = &TreeNode{
			ID:           result.ID,
			Content:      result.Content,
			Embedding:    nil,                // SearchResult 不返回 embedding，需要时再查询
			Level:        pf.config.MaxLevel, // 根节点通常在最高层
			ParentID:     "",
			NodeType:     2, // 根节点
			DocumentID:   result.DocumentID,
			DocumentName: result.DocumentName,
			Section:      result.Section,
		}
	}

	return nodes, nil
}

// traverseDown 从给定节点递归向下查找最相关的子节点路径。
// 返回路径上的所有节点（包括起始节点）。
func (pf *PathFinder) traverseDown(ctx context.Context, currentNode *TreeNode, queryEmbedding []float32) ([]*TreeNode, error) {
	// 检查 context 是否已取消
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context cancelled during traversal: %w", ctx.Err())
	}

	// 初始化路径节点列表，包含当前节点
	pathNodes := []*TreeNode{currentNode}

	// 停止条件：到达叶子节点（Level 0）
	if currentNode.Level == 0 {
		return pathNodes, nil
	}

	// 查找当前节点的子节点
	children, err := pf.findChildren(ctx, currentNode.ID)
	if err != nil {
		logger.Warnw("查找子节点失败",
			"parent_id", currentNode.ID,
			"error", err.Error(),
		)
		// 返回当前路径（不继续向下）
		return pathNodes, nil
	}

	// 如果没有子节点，返回当前路径
	if len(children) == 0 {
		logger.Infow("节点没有子节点，停止向下查找",
			"node_id", currentNode.ID,
			"level", currentNode.Level,
		)
		return pathNodes, nil
	}

	// 选择 topK 个最相关的子节点
	topChildren := pf.selectTopK(children, queryEmbedding, pf.config.TopK)

	logger.Infow("选择最相关子节点",
		"parent_id", currentNode.ID,
		"children_count", len(children),
		"selected_count", len(topChildren),
	)

	// 递归向下查找每个选中的子节点
	for _, child := range topChildren {
		childPath, err := pf.traverseDown(ctx, child, queryEmbedding)
		if err != nil {
			logger.Warnw("递归查找子节点失败",
				"child_id", child.ID,
				"error", err.Error(),
			)
			continue
		}

		// 将子路径添加到当前路径（跳过第一个节点，避免重复）
		if len(childPath) > 1 {
			pathNodes = append(pathNodes, childPath[1:]...)
		}
	}

	return pathNodes, nil
}

// findChildren 查找指定节点的所有子节点。
func (pf *PathFinder) findChildren(ctx context.Context, parentID string) ([]*TreeNode, error) {
	// 构建过滤表达式：parent_id == 'xxx'
	filterExpr := fmt.Sprintf("parent_id == '%s'", parentID)

	// 使用零向量进行查询（主要依赖过滤表达式）
	dummyEmbedding := make([]float32, 768)

	// 查询子节点（topK 设置为较大值，获取所有子节点）
	results, err := pf.store.SearchWithFilter(ctx, pf.config.Collection, dummyEmbedding, filterExpr, 1000)
	if err != nil {
		return nil, fmt.Errorf("查询子节点失败: %w", err)
	}

	// 转换为 TreeNode 格式
	nodes := make([]*TreeNode, len(results))
	for i, result := range results {
		nodes[i] = &TreeNode{
			ID:           result.ID,
			Content:      result.Content,
			Embedding:    nil, // SearchResult 不返回 embedding
			Level:        0,   // 实际 level 从 metadata 读取（如果需要）
			ParentID:     parentID,
			NodeType:     1, // 假设为中间节点
			DocumentID:   result.DocumentID,
			DocumentName: result.DocumentName,
			Section:      result.Section,
		}
	}

	return nodes, nil
}

// selectTopK 从节点列表中选择与查询最相关的 topK 个节点。
// 如果节点数 <= k，返回所有节点。
func (pf *PathFinder) selectTopK(nodes []*TreeNode, queryEmbedding []float32, k int) []*TreeNode {
	// 边界条件：节点数 <= k，返回所有节点
	if len(nodes) <= k {
		return nodes
	}

	// 计算每个节点与查询的相似度
	type nodeSimilarity struct {
		node       *TreeNode
		similarity float64
	}

	similarities := make([]nodeSimilarity, 0, len(nodes))

	for _, node := range nodes {
		// 如果节点没有 embedding，需要重新获取或跳过
		// 这里简化处理：如果没有 embedding，相似度为 0
		var sim float64
		if node.Embedding != nil {
			sim = textutil.CosineSimilarity(queryEmbedding, node.Embedding)
		} else {
			// TODO: 可以选择重新查询 embedding，这里简化为跳过
			sim = 0
		}

		similarities = append(similarities, nodeSimilarity{
			node:       node,
			similarity: sim,
		})
	}

	// 按相似度降序排序
	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].similarity > similarities[j].similarity
	})

	// 选择 topK
	topNodes := make([]*TreeNode, k)
	for i := 0; i < k; i++ {
		topNodes[i] = similarities[i].node
	}

	return topNodes
}

// deduplicateNodes 去除重复节点（基于节点 ID）。
func (pf *PathFinder) deduplicateNodes(nodes []*TreeNode) []*TreeNode {
	seen := make(map[string]bool)
	unique := make([]*TreeNode, 0, len(nodes))

	for _, node := range nodes {
		if !seen[node.ID] {
			seen[node.ID] = true
			unique = append(unique, node)
		}
	}

	return unique
}
