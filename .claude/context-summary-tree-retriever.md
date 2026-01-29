# 项目上下文摘要（TreeRetriever 实现）

**生成时间**：2026-01-24
**任务**：阶段3 - 实现 TreeRetriever 三阶段混合检索
**目标**：验证树形检索能否提升复杂查询准确率 >15%

---

## 1. 相似实现分析

### 实现1：internal/rag/biz/retriever.go（现有向量检索器）

**位置**：`retriever.go:56-147`（Retrieve方法，92行）

**模式**：四阶段检索流程
```go
type Retriever struct {
    store         store.VectorStore
    embedProvider llm.EmbeddingProvider
    enhancer      *enhancer.Enhancer
    config        *RetrieverConfig
}

// 检索流程
func (r *Retriever) Retrieve(ctx context.Context, question string) (*RetrievalResult, error) {
    // 1. 增强查询（查询重写 + HyDE）
    enhancedQuery, embeddings, err := r.enhancer.EnhanceQuery(ctx, question)

    // 2. 执行向量检索（支持多嵌入）
    for _, embedding := range embeddings {
        results, _ := r.store.Search(ctx, r.config.Collection, embedding, r.config.TopK)
        allResults = append(allResults, results...)
    }

    // 3. 重排序结果
    rerankedResults, _ := r.enhancer.RerankResults(ctx, enhancedQuery, allResults)

    // 4. 文档重组（Lost in the Middle策略）
    repackedResults := r.enhancer.RepackDocuments(rerankedResults)

    return &RetrievalResult{Query: enhancedQuery, Results: storeResults}, nil
}
```

**可复用**：
- ✅ `RetrieverConfig` 配置结构（TopK、Collection）
- ✅ `RetrievalResult` 返回结构（Query、Results）
- ✅ `store.VectorStore.SearchWithFilter()` 方法（阶段1已扩展）
- ✅ 错误降级处理模式（增强失败→使用原始查询）

**需注意**：
- Context取消检查：`if ctx.Err() != nil`
- 降级策略：每个步骤失败都有fallback
- 日志记录：每个步骤都记录Infof/Warnw

---

### 实现2：internal/rag/biz/generator.go（答案生成器）

**位置**：`generator.go:33-73`（GenerateAnswer方法，41行）

**模式**：模板化Prompt生成
```go
type Generator struct {
    chatProvider llm.ChatProvider
    config       *GeneratorConfig
}

func (g *Generator) GenerateAnswer(ctx context.Context, question string, results []*store.SearchResult) (*llm.GenerateResponse, error) {
    // 1. 构建上下文（格式化检索结果）
    for i, result := range results {
        contextBuilder.WriteString(fmt.Sprintf("[%d] From %s - %s:\n%s\n\n",
            i+1, result.DocumentName, result.Section, result.Content))
    }

    // 2. 替换提示词模板
    prompt := strings.ReplaceAll(g.config.SystemPrompt, "{{context}}", contextBuilder.String())
    prompt = strings.ReplaceAll(prompt, "{{question}}", question)

    // 3. 调用LLM生成答案
    resp, err := g.chatProvider.Generate(ctx, prompt, "")

    return resp, nil
}
```

**可复用**：
- ✅ 模板化Prompt模式（使用占位符替换）
- ✅ Context取消检查模式
- ✅ 日志记录模式（记录长度和Token数）

**需注意**：
- 空结果处理：返回固定提示而非错误
- Token使用统计：resp.TokenUsage可能为nil

---

### 实现3：internal/rag/biz/tree_builder.go（树构建器）

**位置**：`tree_builder.go:73-140`（BuildTree方法，68行）

**模式**：递归构建 + 并发控制
```go
type TreeBuilder struct {
    store         store.VectorStore
    embedProvider llm.EmbeddingProvider
    chatProvider  llm.ChatProvider
    summarizer    *Summarizer
    clusterer     *KMeansClusterer
    config        *TreeBuilderConfig
}

func (tb *TreeBuilder) BuildTree(ctx context.Context, documentID string) error {
    // 1. 获取叶子节点（Level 0）
    leafNodes, err := tb.getLeafNodes(ctx, documentID)

    // 2. 递归构建层级
    for currentLevel < tb.config.MaxLevel && len(currentNodes) > 5 {
        // KMeans聚类 → 生成摘要 → 创建父节点 → 批量存储
        parentNodes, err := tb.buildLevel(ctx, currentNodes, currentLevel+1)
        currentNodes = parentNodes
        currentLevel++
    }

    // 3. 标记根节点
    for _, node := range currentNodes {
        node.NodeType = 2 // 根节点
    }

    return nil
}
```

**可复用**：
- ✅ `getLeafNodes()` 方法（使用SearchWithFilter查询Level=0）
- ✅ 递归构建模式（停止条件：节点数≤5或达到max_level）
- ✅ pool.BackgroundPool 并发控制（带降级处理）

**需注意**：
- 异步池时序问题：SubmitToType不等待任务完成
- 批量存储：使用storeNodes批量插入
- 停止条件：节点数≤5时标记为根节点

---

## 2. 项目约定

### 命名约定
- **接口**：以`er`结尾（Retriever、Builder、Provider）
- **配置**：以`Config`结尾（RetrieverConfig、TreeBuilderConfig）
- **结果**：以`Result`结尾（RetrievalResult、SearchResult）
- **变量**：驼峰命名（embedProvider、chatProvider）

### 文件组织
- **业务逻辑**：`internal/rag/biz/` - Retriever、Generator、TreeBuilder
- **存储层**：`internal/rag/store/` - VectorStore接口、MilvusStore实现
- **工具层**：`internal/pkg/rag/` - textutil、enhancer
- **通用库**：`pkg/llm/` - ChatProvider、EmbeddingProvider接口

### 导入顺序
```go
import (
    // 1. 标准库
    "context"
    "fmt"

    // 2. 第三方库
    "github.com/kart-io/logger"

    // 3. 项目内部库
    "github.com/kart-io/sentinel-x/internal/rag/store"
    "github.com/kart-io/sentinel-x/pkg/llm"
)
```

### 代码风格
- ✅ 简体中文注释、英文代码
- ✅ gofmt 格式化
- ✅ 错误包装：`fmt.Errorf("xxx失败: %w", err)`
- ✅ 日志记录：logger.Infof/Warnw/Errorf

---

## 3. 可复用组件清单

### VectorStore 接口（已扩展）
```go
// internal/rag/store/store.go
type VectorStore interface {
    // 基础检索
    Search(ctx context.Context, collection string, embedding []float32, topK int) ([]*SearchResult, error)

    // 过滤检索（阶段1已扩展）
    SearchWithFilter(ctx context.Context, collection string, embedding []float32, expr string, topK int) ([]*SearchResult, error)

    // 其他方法...
}
```

**用途**：TreeRetriever将使用SearchWithFilter查询不同层级的节点

### TreeNode 数据结构（阶段2已定义）
```go
// internal/rag/biz/cluster.go
type TreeNode struct {
    ID           string
    Content      string
    Embedding    []float32
    Level        int    // 0=叶子，1+=中间/根
    ParentID     string
    NodeType     int    // 0=叶子，1=中间，2=根
    DocumentID   string
    DocumentName string
    Section      string
}
```

**用途**：PathFinder和TreeRetriever将操作此数据结构

### textutil.CosineSimilarity（已有）
```go
// internal/pkg/rag/textutil/textutil.go:15-34
func CosineSimilarity(a, b []float32) float64
```

**用途**：计算查询向量与节点向量的相似度

### logger（统一日志）
```go
// github.com/kart-io/logger
logger.Infof(format, args...)
logger.Warnw(msg, keysAndValues...)
logger.Errorf(format, args...)
```

**用途**：所有日志记录

---

## 4. 测试策略

### 测试框架
```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)
```

### 测试模式

**表驱动测试**（参考：cluster_test.go）
```go
func TestTreeRetriever_Retrieve_Scenarios(t *testing.T) {
    tests := []struct {
        name         string
        query        string
        expectTopK   int
        expectLevels []int
    }{
        {name: "简单查询", query: "测试问题", expectTopK: 10, expectLevels: []int{0}},
        {name: "复杂查询", query: "复杂问题", expectTopK: 20, expectLevels: []int{0, 1, 2}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑
        })
    }
}
```

**Mock实现**（参考：tree_builder_test.go）
```go
// mockVectorStore
type mockVectorStore struct {
    searchResults []*store.SearchResult
    searchError   error
}

func (m *mockVectorStore) SearchWithFilter(ctx, collection, embedding, expr, topK) ([]*SearchResult, error) {
    if m.searchError != nil {
        return nil, m.searchError
    }
    return m.searchResults, nil
}

// 接口实现验证（编译时检查）
var _ store.VectorStore = (*mockVectorStore)(nil)
```

**辅助函数**（参考：cache_test.go）
```go
// setupTestRetriever 创建测试用检索器
func setupTestRetriever(t *testing.T) *TreeRetriever {
    mockStore := &mockVectorStore{...}
    mockEmbed := &mockEmbeddingProvider{...}
    config := &TreeRetrieverConfig{...}
    return NewTreeRetriever(mockStore, mockEmbed, config)
}
```

### 覆盖要求
- ✅ 单元测试覆盖率 >80%
- ✅ 核心方法必须测试：Retrieve、FindPaths、RankNodes
- ✅ 边界条件测试：空结果、单节点、多层级
- ✅ 错误处理测试：SearchWithFilter失败、向量生成失败

---

## 5. 依赖和集成点

### 外部依赖
- **VectorStore**：`internal/rag/store.VectorStore` - 向量存储接口
- **EmbeddingProvider**：`pkg/llm.EmbeddingProvider` - 向量生成
- **Logger**：`github.com/kart-io/logger` - 日志记录

### 内部依赖
- **TreeNode**：`internal/rag/biz.TreeNode` - 树节点数据结构（阶段2已定义）
- **textutil**：`internal/pkg/rag/textutil` - 余弦相似度计算

### 集成方式
- **构造函数注入**：`NewTreeRetriever(store, embedProvider, config)`
- **接口调用**：通过VectorStore接口操作数据
- **配置驱动**：通过TreeRetrieverConfig控制行为

### 配置来源
```yaml
# configs/rag.yaml
rag:
  tree:
    enabled: false           # POC开关（默认关闭）
    max-level: 3            # 树最大层级
    num-clusters: 5         # 每层聚类数
    retriever:
      top-k-path: 3         # 路径查找topK
      top-k-leaf: 20        # 叶子检索topK
      score-weight-sim: 0.7 # 相似度权重
      score-weight-level: 0.3 # 层级权重
```

---

## 6. 技术选型理由

### 为什么用三阶段混合检索？

**阶段1：自顶向下路径查找**
- **理由**：捕捉全局主题和层级语义
- **优势**：相比单纯向量检索，能理解文档的宏观结构
- **实现**：从根节点递归向下，选择最相关子节点

**阶段2：叶子层全局检索**
- **理由**：保留细节信息，弥补路径查找的遗漏
- **优势**：确保召回率，避免只依赖路径导致遗漏
- **实现**：在Level=0执行向量检索，过滤已有节点

**阶段3：综合排序**
- **理由**：平衡相似度和层级重要性
- **优势**：高层节点（摘要）和低层节点（细节）各有价值
- **实现**：评分 = 相似度×0.7 + 层级权重×0.3

### 为什么用递归路径查找？
- **理由**：树形结构天然适合递归
- **优势**：代码简洁、逻辑清晰
- **风险**：深度过大时可能栈溢出
- **缓解**：限制max_level=3，确保深度可控

### 为什么用余弦相似度？
- **理由**：与向量检索保持一致
- **优势**：归一化后不受向量长度影响
- **实现**：复用textutil.CosineSimilarity

---

## 7. 关键风险点

### 并发问题
- **风险**：路径查找和叶子检索可能并发访问VectorStore
- **缓解**：VectorStore接口设计为线程安全（由实现保证）
- **注意**：不使用pool.BackgroundPool（检索是同步操作）

### 边界条件
- **空树**：文档没有树索引（tree.enabled=false或构建失败）
  - **处理**：降级到纯叶子检索（相当于向量检索）
- **单节点树**：只有根节点（文档太小）
  - **处理**：直接返回根节点内容
- **查询向量生成失败**
  - **处理**：返回错误，不降级（与现有Retriever保持一致）

### 性能瓶颈
- **SearchWithFilter调用次数**：路径查找每层调用1次，叶子检索调用1次
  - **优化**：限制max_level=3，最多4次调用
- **综合排序复杂度**：O(n log n)，n为结果总数
  - **优化**：限制总结果数≤30（top-k-path=3, top-k-leaf=20）

### 安全考虑
- **无**：检索操作只读，无安全风险
- **注意**：expr过滤表达式由代码硬编码（不接受用户输入）

---

## 8. 实现清单（待完成）

### Task #1：实现 PathFinder（2天）
**文件**：`internal/rag/biz/path_finder.go`（新建，~150行）

**核心方法**：
```go
type PathFinder struct {
    store         store.VectorStore
    config        *PathFinderConfig
}

// FindPaths 自顶向下查找路径
func (pf *PathFinder) FindPaths(ctx context.Context, queryEmbedding []float32, documentID string, topK int) ([]*TreeNode, error)

// traverseDown 递归向下查找
func (pf *PathFinder) traverseDown(ctx context.Context, parentID string, queryEmbedding []float32, level int) ([]*TreeNode, error)
```

**测试**：`path_finder_test.go`（~200行）

---

### Task #2：实现 TreeRetriever（2-3天）
**文件**：`internal/rag/biz/tree_retriever.go`（新建，~200行）

**核心方法**：
```go
type TreeRetriever struct {
    store         store.VectorStore
    embedProvider llm.EmbeddingProvider
    pathFinder    *PathFinder
    config        *TreeRetrieverConfig
}

// Retrieve 三阶段混合检索
func (tr *TreeRetriever) Retrieve(ctx context.Context, question string) (*RetrievalResult, error)

// retrieveLeafNodes 叶子层检索
func (tr *TreeRetriever) retrieveLeafNodes(ctx context.Context, queryEmbedding []float32, excludeIDs []string) ([]*TreeNode, error)

// rankAndMerge 综合排序和去重
func (tr *TreeRetriever) rankAndMerge(pathNodes, leafNodes []*TreeNode, queryEmbedding []float32) []*store.SearchResult
```

**测试**：`tree_retriever_test.go`（~250行）

---

### Task #3：实现综合排序算法（1天）
**位置**：`tree_retriever.go` 中的 `rankAndMerge` 方法

**评分公式**：
```go
// 计算综合评分
score := cosineSimilarity(queryEmbedding, node.Embedding) * 0.7 + levelWeight(node.Level) * 0.3

// 层级权重
func levelWeight(level int) float32 {
    // Level 0（叶子）: 0.3
    // Level 1（中间）: 0.6
    // Level 2+（根）: 1.0
    return min(1.0, 0.3 + float32(level)*0.3)
}
```

---

### Task #4：编写单元测试（2天）
**文件**：
- `path_finder_test.go`（~200行）
- `tree_retriever_test.go`（~250行）

**测试用例**：
- PathFinder：单层查找、多层递归、边界条件
- TreeRetriever：完整检索流程、结果融合、排序验证
- Mock：VectorStore、EmbeddingProvider

---

### Task #5：集成测试和验证（1-2天）
**验证项**：
- ✅ 与现有Retriever共存
- ✅ 配置开关正常工作
- ✅ 检索延迟可接受（<2秒）
- ✅ 结果质量提升验证

---

## 9. 上下文充分性验证

### ✅ 全部7项检查通过

1. ✅ **相似实现**：retriever.go、generator.go、tree_builder.go
2. ✅ **实现模式**：依赖注入 + 接口隔离 + 错误降级
3. ✅ **可复用组件**：VectorStore、TreeNode、textutil、logger
4. ✅ **命名约定**：驼峰、接口er结尾、配置Config结尾
5. ✅ **测试策略**：testify/assert + 表驱动 + Mock实现
6. ✅ **无重复造轮子**：TreeRetriever是新功能
7. ✅ **依赖和集成**：VectorStore、EmbeddingProvider、配置驱动

---

## 10. 下一步行动

**立即开始**：编写 PathFinder 实现

**文件清单**：
1. `internal/rag/biz/path_finder.go`（新建）
2. `internal/rag/biz/path_finder_test.go`（新建）
3. `internal/rag/biz/tree_retriever.go`（新建）
4. `internal/rag/biz/tree_retriever_test.go`（新建）

**预计时间**：7-10天

**验收标准**：
- ✅ 单元测试覆盖率 >80%
- ✅ 测试通过率 100%
- ✅ 检索延迟 P95 <2秒
- ✅ 代码编译通过，无破坏性影响

---

**状态**：上下文收集完成，准备开始编码实现
**最后更新**：2026-01-24
