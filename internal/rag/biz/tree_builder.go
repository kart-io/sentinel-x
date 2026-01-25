package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/internal/rag/store"
	"github.com/kart-io/sentinel-x/pkg/infra/pool"
	"github.com/kart-io/sentinel-x/pkg/llm"
)

// TreeBuilderConfig 树构建器配置。
type TreeBuilderConfig struct {
	// MaxLevel 树的最大层级（0=叶子层）。
	MaxLevel int
	// NumClusters 每层的聚类数量。
	NumClusters int
	// Collection Milvus 集合名称。
	Collection string
	// SummaryMaxTokens 摘要最大 token 长度。
	SummaryMaxTokens int
	// SummaryModel 摘要生成使用的模型名称。
	SummaryModel string
}

// TreeBuilder 负责构建文档的树形索引。
// 使用 KMeans 聚类和摘要生成递归构建多层树结构。
type TreeBuilder struct {
	store         store.VectorStore
	embedProvider llm.EmbeddingProvider
	chatProvider  llm.ChatProvider
	summarizer    *Summarizer
	clusterer     *KMeansClusterer
	config        *TreeBuilderConfig
}

// NewTreeBuilder 创建树构建器实例。
func NewTreeBuilder(
	vectorStore store.VectorStore,
	embedProvider llm.EmbeddingProvider,
	chatProvider llm.ChatProvider,
	config *TreeBuilderConfig,
) *TreeBuilder {
	// 创建 Summarizer
	summarizerConfig := &SummarizerConfig{
		MaxTokens: config.SummaryMaxTokens,
		Model:     config.SummaryModel,
	}
	summarizer := NewSummarizer(chatProvider, summarizerConfig)

	// 创建 KMeansClusterer
	clustererConfig := &KMeansClustererConfig{
		MaxIterations:        10,
		ConvergenceThreshold: 0.001,
	}
	clusterer := NewKMeansClusterer(clustererConfig)

	return &TreeBuilder{
		store:         vectorStore,
		embedProvider: embedProvider,
		chatProvider:  chatProvider,
		summarizer:    summarizer,
		clusterer:     clusterer,
		config:        config,
	}
}

// BuildTree 为文档构建树形索引。
// documentID 为文档的唯一标识。
// 该方法会从已存储的叶子节点（Level 0）开始，递归构建多层树结构。
func (tb *TreeBuilder) BuildTree(ctx context.Context, documentID string) error {
	logger.Infow("开始构建树形索引",
		"document_id", documentID,
		"max_level", tb.config.MaxLevel,
		"num_clusters", tb.config.NumClusters,
	)

	// 1. 获取叶子节点（Level 0）
	leafNodes, err := tb.getLeafNodes(ctx, documentID)
	if err != nil {
		return fmt.Errorf("获取叶子节点失败: %w", err)
	}

	if len(leafNodes) == 0 {
		return fmt.Errorf("文档 %s 没有叶子节点", documentID)
	}

	logger.Infow("获取叶子节点完成",
		"document_id", documentID,
		"leaf_count", len(leafNodes),
	)

	// 2. 递归构建层级
	currentNodes := leafNodes
	currentLevel := 0

	for currentLevel < tb.config.MaxLevel && len(currentNodes) > 5 {
		logger.Infow("开始构建层级",
			"level", currentLevel+1,
			"parent_count", len(currentNodes),
		)

		// 构建下一层
		parentNodes, err := tb.buildLevel(ctx, currentNodes, currentLevel+1)
		if err != nil {
			return fmt.Errorf("构建层级 %d 失败: %w", currentLevel+1, err)
		}

		logger.Infow("层级构建完成",
			"level", currentLevel+1,
			"node_count", len(parentNodes),
		)

		// 准备下一次迭代
		currentNodes = parentNodes
		currentLevel++
	}

	// 3. 标记根节点
	if len(currentNodes) <= 5 {
		for _, node := range currentNodes {
			node.NodeType = 2 // 根节点
		}

		// 存储根节点
		if err := tb.storeNodes(ctx, currentNodes); err != nil {
			return fmt.Errorf("存储根节点失败: %w", err)
		}

		logger.Infow("树形索引构建完成",
			"document_id", documentID,
			"max_level", currentLevel,
			"root_count", len(currentNodes),
		)
	}

	return nil
}

// buildLevel 构建一个层级的节点。
// 对当前层节点进行聚类，为每个簇生成摘要和父节点。
func (tb *TreeBuilder) buildLevel(ctx context.Context, nodes []*TreeNode, level int) ([]*TreeNode, error) {
	// 1. 聚类
	clusters, err := tb.clusterer.Cluster(nodes, tb.config.NumClusters)
	if err != nil {
		return nil, fmt.Errorf("聚类失败: %w", err)
	}

	logger.Infow("聚类完成",
		"level", level,
		"cluster_count", len(clusters),
	)

	// 2. 为每个簇创建父节点
	parentNodes := make([]*TreeNode, 0, len(clusters))

	for i, cluster := range clusters {
		if len(cluster) == 0 {
			continue // 跳过空簇
		}

		// 创建父节点
		parentNode, err := tb.createParentNode(ctx, cluster, level)
		if err != nil {
			logger.Warnw("创建父节点失败，跳过该簇",
				"level", level,
				"cluster_index", i,
				"error", err.Error(),
			)
			continue
		}

		// 更新子节点的 ParentID
		for _, child := range cluster {
			child.ParentID = parentNode.ID
		}

		parentNodes = append(parentNodes, parentNode)

		logger.Infow("父节点创建完成",
			"level", level,
			"cluster_index", i,
			"children_count", len(cluster),
			"parent_id", parentNode.ID,
		)
	}

	// 3. 批量存储父节点和更新后的子节点
	if err := tb.storeNodes(ctx, parentNodes); err != nil {
		return nil, fmt.Errorf("存储父节点失败: %w", err)
	}

	// 更新子节点的 ParentID（需要重新存储）
	allChildren := make([]*TreeNode, 0)
	for _, cluster := range clusters {
		allChildren = append(allChildren, cluster...)
	}
	if err := tb.storeNodes(ctx, allChildren); err != nil {
		logger.Warnw("更新子节点 ParentID 失败",
			"level", level,
			"error", err.Error(),
		)
	}

	return parentNodes, nil
}

// createParentNode 为一组子节点创建父节点。
func (tb *TreeBuilder) createParentNode(ctx context.Context, children []*TreeNode, level int) (*TreeNode, error) {
	if len(children) == 0 {
		return nil, fmt.Errorf("子节点列表为空")
	}

	// 1. 收集子节点的内容
	contents := make([]string, len(children))
	for i, child := range children {
		contents[i] = child.Content
	}

	// 2. 使用 pool.BackgroundPool 生成摘要（带降级处理）
	var summary string
	var summaryErr error

	summaryTask := func() {
		summary, summaryErr = tb.summarizer.Summarize(ctx, contents)
	}

	// 提交到后台池
	err := pool.SubmitToType(pool.BackgroundPool, summaryTask)
	if err != nil {
		// 降级：直接在当前 goroutine 执行
		logger.Warnw("后台池不可用，降级到同步执行",
			"error", err.Error(),
		)
		summaryTask()
	}

	if summaryErr != nil {
		return nil, fmt.Errorf("生成摘要失败: %w", summaryErr)
	}

	// 3. 生成摘要的向量
	embedding, err := tb.embedProvider.EmbedSingle(ctx, summary)
	if err != nil {
		return nil, fmt.Errorf("生成向量失败: %w", err)
	}

	// 4. 创建父节点
	// 使用第一个子节点的文档信息
	//nolint:gosec // G602: children 非空已在函数开头检查（第212行）
	firstChild := children[0]

	parentNode := &TreeNode{
		ID:           fmt.Sprintf("%s_L%d_%d", firstChild.DocumentID, level, len(parentNodes)),
		Content:      summary,
		Embedding:    embedding,
		Level:        level,
		ParentID:     "", // 稍后设置
		NodeType:     1,  // 中间节点
		DocumentID:   firstChild.DocumentID,
		DocumentName: firstChild.DocumentName,
		Section:      firstChild.Section,
	}

	return parentNode, nil
}

// storeNodes 批量存储节点到 VectorStore。
// 适配 Milvus Serverless 速率限制（0.1 req/s），采用分批写入策略。
func (tb *TreeBuilder) storeNodes(ctx context.Context, nodes []*TreeNode) error {
	if len(nodes) == 0 {
		return nil
	}

	// 转换为 store.Chunk 格式
	chunks := make([]*store.Chunk, len(nodes))
	for i, node := range nodes {
		chunks[i] = &store.Chunk{
			ID:           node.ID,
			DocumentID:   node.DocumentID,
			DocumentName: node.DocumentName,
			Section:      node.Section,
			Content:      node.Content,
			Embedding:    node.Embedding,
			Level:        node.Level,
			ParentID:     node.ParentID,
			NodeType:     node.NodeType,
		}
	}

	// 批次处理：适配 Milvus Serverless 速率限制
	// Serverless 免费版限制：0.1 req/s（每10秒1个请求）
	const batchSize = 10                         // 每批10个节点
	const delayBetweenBatches = 12 * time.Second // 批次间延迟12秒

	totalBatches := (len(chunks) + batchSize - 1) / batchSize
	logger.Infow("开始分批存储节点",
		"total_nodes", len(chunks),
		"batch_size", batchSize,
		"total_batches", totalBatches,
		"estimated_time_seconds", totalBatches*12,
	)

	var allIDs []string
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]
		batchNum := i/batchSize + 1

		logger.Infow("正在插入批次",
			"batch", batchNum,
			"total_batches", totalBatches,
			"batch_size", len(batch),
		)

		// 批量插入当前批次
		ids, err := tb.store.Insert(ctx, tb.config.Collection, batch)
		if err != nil {
			return fmt.Errorf("批次 %d/%d 插入失败: %w", batchNum, totalBatches, err)
		}

		allIDs = append(allIDs, ids...)

		logger.Infow("批次插入完成",
			"batch", batchNum,
			"inserted", len(ids),
		)

		// 非最后一批，等待以避免速率限制
		if end < len(chunks) {
			logger.Infow("等待速率限制窗口",
				"delay_seconds", int(delayBetweenBatches.Seconds()),
				"remaining_batches", totalBatches-batchNum,
			)
			time.Sleep(delayBetweenBatches)
		}
	}

	logger.Infow("所有节点存储完成",
		"total_count", len(nodes),
		"total_ids", len(allIDs),
	)

	return nil
}

// getLeafNodes 获取文档的所有叶子节点（Level 0）。
// 注意：这里假设叶子节点已经通过 Indexer 创建并存储。
func (tb *TreeBuilder) getLeafNodes(ctx context.Context, documentID string) ([]*TreeNode, error) {
	// 使用 SearchWithFilter 查询指定文档的叶子节点
	// 注意：Milvus Serverless 限制 topK <= 1024，因此需要分批查询

	// 临时方案：使用零向量进行查询（因为我们主要依赖过滤表达式）
	dummyEmbedding := make([]float32, 768) // 假设向量维度为 768

	// 构建过滤表达式：根据 documentID 是否为空决定过滤条件
	var filterExpr string
	if documentID == "" {
		// POC 阶段：为所有文档构建树，只过滤 level == 0
		filterExpr = "level == 0"
		logger.Infow("获取所有文档的叶子节点", "filter", filterExpr)
	} else {
		// 为特定文档构建树：document_id == 'xxx' AND level == 0
		filterExpr = fmt.Sprintf("document_id == '%s' && level == 0", documentID)
		logger.Infow("获取指定文档的叶子节点", "document_id", documentID, "filter", filterExpr)
	}

	// 适配 Milvus Serverless 限制：topK 最大 1024
	// 策略：使用最大允许值 1024，通常足够获取单个文档的所有叶子节点
	const maxTopK = 1024
	results, err := tb.store.SearchWithFilter(ctx, tb.config.Collection, dummyEmbedding, filterExpr, maxTopK)
	if err != nil {
		return nil, fmt.Errorf("查询叶子节点失败: %w", err)
	}

	// 如果返回结果达到 maxTopK，可能还有更多节点未返回（需要分页）
	// 但对于单个文档，通常不会超过 1024 个叶子节点
	if len(results) == maxTopK {
		logger.Warnw("叶子节点数量达到查询上限，可能存在未获取的节点",
			"document_id", documentID,
			"retrieved", len(results),
			"limit", maxTopK,
		)
	}

	logger.Infow("获取叶子节点完成",
		"document_id", documentID,
		"leaf_count", len(results),
	)

	// 转换为 TreeNode 格式
	nodes := make([]*TreeNode, len(results))
	for i, result := range results {
		nodes[i] = &TreeNode{
			ID:           result.ID,
			Content:      result.Content,
			Embedding:    nil, // SearchResult 不返回 embedding，需要的话可以再查询
			Level:        0,
			ParentID:     "",
			NodeType:     0, // 叶子节点
			DocumentID:   result.DocumentID,
			DocumentName: result.DocumentName,
			Section:      result.Section,
		}
	}

	return nodes, nil
}

// 用于生成父节点 ID 的计数器（简化实现）
var parentNodes []string
