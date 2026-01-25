package biz

import (
	"context"
	"fmt"
	"sort"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/pkg/rag/textutil"
	"github.com/kart-io/sentinel-x/internal/rag/store"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// TreeRetrieverConfig 树形检索器配置。
type TreeRetrieverConfig struct {
	// Collection 集合名称。
	Collection string
	// TopKPath 路径查找返回的节点数量。
	TopKPath int
	// TopKLeaf 叶子层检索返回的节点数量。
	TopKLeaf int
	// ScoreWeightSim 相似度权重（0-1）。
	ScoreWeightSim float64
	// ScoreWeightLevel 层级权重（0-1）。
	ScoreWeightLevel float64
	// MaxLevel 树的最大层级。
	MaxLevel int
}

// TreeRetriever 负责在树形索引中执行三阶段混合检索。
// 阶段1：自顶向下路径查找
// 阶段2：叶子层全局检索
// 阶段3：综合排序和去重
type TreeRetriever struct {
	store         store.VectorStore
	embedProvider llm.EmbeddingProvider
	pathFinder    *PathFinder
	config        *TreeRetrieverConfig
}

// NewTreeRetriever 创建树形检索器实例。
func NewTreeRetriever(
	vectorStore store.VectorStore,
	embedProvider llm.EmbeddingProvider,
	config *TreeRetrieverConfig,
) *TreeRetriever {
	// 创建 PathFinder
	pathFinderConfig := &PathFinderConfig{
		Collection: config.Collection,
		TopK:       config.TopKPath,
		MaxLevel:   config.MaxLevel,
	}
	pathFinder := NewPathFinder(vectorStore, pathFinderConfig)

	return &TreeRetriever{
		store:         vectorStore,
		embedProvider: embedProvider,
		pathFinder:    pathFinder,
		config:        config,
	}
}

// Retrieve 执行三阶段混合检索。
// 返回综合排序后的检索结果。
func (tr *TreeRetriever) Retrieve(ctx context.Context, question string, documentID string) (*RetrievalResult, error) {
	logger.Infow("开始树形检索",
		"question", question,
		"document_id", documentID,
	)

	// 检查 context 是否已取消
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context cancelled before retrieval: %w", ctx.Err())
	}

	// 1. 生成查询向量
	logger.Info("阶段0: 生成查询向量...")
	queryEmbedding, err := tr.embedProvider.EmbedSingle(ctx, question)
	if err != nil {
		return nil, fmt.Errorf("生成查询向量失败: %w", err)
	}

	// 2. 阶段1：自顶向下路径查找
	logger.Info("阶段1: 自顶向下路径查找...")
	pathNodes, err := tr.pathFinder.FindPaths(ctx, queryEmbedding, documentID)
	if err != nil {
		logger.Warnw("路径查找失败，降级到纯叶子检索",
			"error", err.Error(),
		)
		pathNodes = []*TreeNode{}
	}

	logger.Infow("路径查找完成",
		"path_nodes_count", len(pathNodes),
	)

	// 3. 阶段2：叶子层全局检索
	logger.Info("阶段2: 叶子层全局检索...")

	// 提取已在路径中的节点ID（用于去重）
	pathNodeIDs := make(map[string]bool)
	for _, node := range pathNodes {
		pathNodeIDs[node.ID] = true
	}

	// 执行叶子层检索
	leafNodes, err := tr.retrieveLeafNodes(ctx, queryEmbedding, documentID, pathNodeIDs)
	if err != nil {
		logger.Warnw("叶子层检索失败",
			"error", err.Error(),
		)
		// 如果叶子检索也失败，只使用路径节点
		leafNodes = []*TreeNode{}
	}

	logger.Infow("叶子层检索完成",
		"leaf_nodes_count", len(leafNodes),
	)

	// 4. 阶段3：综合排序和去重
	logger.Info("阶段3: 综合排序和去重...")

	// 如果路径查找和叶子检索都失败，返回空结果
	if len(pathNodes) == 0 && len(leafNodes) == 0 {
		logger.Warn("路径查找和叶子检索都未返回结果")
		return &RetrievalResult{
			Query:   question,
			Results: []*store.SearchResult{},
		}, nil
	}

	// 综合排序
	finalResults := tr.rankAndMerge(pathNodes, leafNodes, queryEmbedding)

	logger.Infow("树形检索完成",
		"total_results", len(finalResults),
	)

	return &RetrievalResult{
		Query:   question,
		Results: finalResults,
	}, nil
}

// retrieveLeafNodes 执行叶子层全局检索。
// 查询 Level=0 的节点，并排除已在路径中的节点。
func (tr *TreeRetriever) retrieveLeafNodes(
	ctx context.Context,
	queryEmbedding []float32,
	documentID string,
	excludeIDs map[string]bool,
) ([]*TreeNode, error) {
	// 构建过滤表达式：处理空 documentID
	var filterExpr string
	if documentID == "" {
		// POC 阶段：查询所有文档的叶子节点
		filterExpr = "level == 0"
		logger.Infow("叶子层检索（所有文档）", "filter", filterExpr)
	} else {
		// 查询指定文档的叶子节点
		filterExpr = fmt.Sprintf("document_id == '%s' && level == 0", documentID)
		logger.Infow("叶子层检索（指定文档）", "document_id", documentID, "filter", filterExpr)
	}

	// 使用 SearchWithFilter 查询叶子节点
	results, err := tr.store.SearchWithFilter(
		ctx,
		tr.config.Collection,
		queryEmbedding,
		filterExpr,
		tr.config.TopKLeaf,
	)
	if err != nil {
		return nil, fmt.Errorf("叶子层检索失败: %w", err)
	}

	// 转换为 TreeNode 并过滤已在路径中的节点
	nodes := make([]*TreeNode, 0, len(results))
	for _, result := range results {
		// 跳过已在路径中的节点
		if excludeIDs[result.ID] {
			continue
		}

		nodes = append(nodes, &TreeNode{
			ID:           result.ID,
			Content:      result.Content,
			Embedding:    nil, // SearchResult 不返回 embedding
			Level:        0,   // 叶子节点
			ParentID:     "",
			NodeType:     0, // 叶子节点
			DocumentID:   result.DocumentID,
			DocumentName: result.DocumentName,
			Section:      result.Section,
		})
	}

	return nodes, nil
}

// rankAndMerge 综合排序和去重。
// 评分公式：score = cosineSimilarity × ScoreWeightSim + levelWeight × ScoreWeightLevel
func (tr *TreeRetriever) rankAndMerge(
	pathNodes []*TreeNode,
	leafNodes []*TreeNode,
	queryEmbedding []float32,
) []*store.SearchResult {
	// 合并所有节点
	allNodes := make([]*TreeNode, 0, len(pathNodes)+len(leafNodes))
	allNodes = append(allNodes, pathNodes...)
	allNodes = append(allNodes, leafNodes...)

	// 去重（基于节点ID）
	uniqueNodes := tr.deduplicateNodes(allNodes)

	// 计算综合评分
	type nodeScore struct {
		node  *TreeNode
		score float64
	}

	nodeScores := make([]nodeScore, 0, len(uniqueNodes))

	for _, node := range uniqueNodes {
		// 计算相似度评分
		var simScore float64
		if node.Embedding != nil {
			simScore = textutil.CosineSimilarity(queryEmbedding, node.Embedding)
		} else {
			// 如果节点没有 embedding，使用默认值
			// TODO: 可以选择重新查询 embedding
			simScore = 0.5 // 中等相似度
		}

		// 计算层级权重
		levelScore := tr.calculateLevelWeight(node.Level)

		// 综合评分
		finalScore := simScore*tr.config.ScoreWeightSim + levelScore*tr.config.ScoreWeightLevel

		nodeScores = append(nodeScores, nodeScore{
			node:  node,
			score: finalScore,
		})
	}

	// 按评分降序排序
	sort.Slice(nodeScores, func(i, j int) bool {
		return nodeScores[i].score > nodeScores[j].score
	})

	// 转换为 store.SearchResult
	results := make([]*store.SearchResult, len(nodeScores))
	for i, ns := range nodeScores {
		results[i] = &store.SearchResult{
			ID:           ns.node.ID,
			DocumentID:   ns.node.DocumentID,
			DocumentName: ns.node.DocumentName,
			Section:      ns.node.Section,
			Content:      ns.node.Content,
			Score:        float32(ns.score),
			// 添加树形索引元数据
			Metadata: map[string]interface{}{
				"tree_level":   ns.node.Level,
				"node_type":    ns.node.NodeType,
				"parent_id":    ns.node.ParentID,
				"final_score":  ns.score,
				"level_weight": tr.calculateLevelWeight(ns.node.Level),
			},
		}
	}

	logger.Infow("树形检索排序完成",
		"total_results", len(results),
		"path_nodes", len(pathNodes),
		"leaf_nodes", len(leafNodes),
	)

	return results
}

// calculateLevelWeight 计算层级权重。
// Level 0（叶子）: 0.3
// Level 1（中间）: 0.6
// Level 2+（根）: 1.0
func (tr *TreeRetriever) calculateLevelWeight(level int) float64 {
	baseWeight := 0.3
	increment := 0.3

	weight := baseWeight + float64(level)*increment

	// 限制最大值为 1.0
	if weight > 1.0 {
		weight = 1.0
	}

	return weight
}

// deduplicateNodes 去除重复节点（基于节点 ID）。
func (tr *TreeRetriever) deduplicateNodes(nodes []*TreeNode) []*TreeNode {
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
