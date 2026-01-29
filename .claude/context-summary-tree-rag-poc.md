# Tree-based RAG POC 项目上下文摘要

**生成时间**：2026-01-24
**任务名称**：Tree-based RAG (RAPTOR) POC 实施
**目标**：验证树形索引能否提升复杂查询准确率 >15%

---

## 1. 相似实现分析

### 实现1: internal/rag/biz/service.go (RAG服务编排)
- **文件位置**：`internal/rag/biz/service.go:25-189`
- **实现模式**：依赖注入 + 接口隔离
- **核心结构**：
  ```go
  type RAGService struct {
      indexer       *Indexer
      retriever     *Retriever
      generator     *Generator
      cache         *QueryCache
      store         store.VectorStore
      embedProvider llm.EmbeddingProvider
      chatProvider  llm.ChatProvider
      collection    string
      metrics       *metrics.RAGMetrics
  }
  ```
- **可复用模式**：
  - 构造函数依赖注入：`NewRAGService(vectorStore, embedProvider, chatProvider, cache, config)`
  - 接口抽象：VectorStore, EmbeddingProvider, ChatProvider
  - 配置结构体：ServiceConfig 封装所有配置
  - 错误处理：defer 记录指标，fmt.Errorf 包装错误
  - 非侵入式扩展：通过配置开关控制功能启用
- **需注意事项**：
  - Query 方法中集成了缓存、检索、生成三个阶段
  - 使用 defer 确保指标记录（成功和失败都记录）
  - 缓存失败不阻塞主流程（记录警告即可）

### 实现2: internal/rag/biz/indexer.go (文档索引)
- **文件位置**：`internal/rag/biz/indexer.go:32-193`
- **实现模式**：批量处理 + 错误容忍
- **核心逻辑**：
  ```go
  // 分块逻辑可复用
  func (i *Indexer) parseAndChunk(content, docID, docName string) []*store.Chunk {
      // 1. 按标题分割文档（使用正则 ^#{1,6}\s+）
      // 2. 每个 section 使用 textutil.SplitIntoChunks 分块
      // 3. 过滤小于20字符的块
      // 4. 清理和截断文本（SanitizeUTF8, TruncateString）
  }
  ```
- **可复用组件**：
  - `textutil.SplitIntoChunks(text, chunkSize, overlap)` - 文本分块
  - `textutil.SanitizeUTF8(text)` - 清理无效UTF-8字符
  - `textutil.TruncateString(text, maxLen)` - 截断字符串
  - `textutil.HashString(text)` - 生成文档ID
- **需注意事项**：
  - 批量处理（batchSize=10），失败不影响其他批次
  - 使用 logger.Warnf 记录非致命错误
  - 嵌入向量生成在批量插入前完成（批量调用 embedProvider.Embed）

### 实现3: internal/rag/store/milvus.go (Milvus集成)
- **文件位置**：`internal/rag/store/milvus.go:11-112`
- **实现模式**：接口实现 + Schema定义
- **Schema 定义方式**：
  ```go
  func (s *MilvusStore) CreateCollection(ctx context.Context, config *CollectionConfig) error {
      schema := &milvus.CollectionSchema{
          Name:        config.Name,
          Description: config.Description,
          Dimension:   config.Dimension,
          MetaFields: []milvus.MetaField{
              {Name: "document_id", DataType: entity.FieldTypeVarChar, MaxLen: 64},
              {Name: "document_name", DataType: entity.FieldTypeVarChar, MaxLen: 255},
              {Name: "section", DataType: entity.FieldTypeVarChar, MaxLen: 255},
              {Name: "content", DataType: entity.FieldTypeVarChar, MaxLen: 65535},
          },
      }
      return s.client.CreateCollection(ctx, schema)
  }
  ```
- **扩展点**：
  - 在 MetaFields 中添加新字段：`level`, `parent_id`, `node_type`
  - Insert 方法需要扩展 metadata 映射
  - Search 方法需要添加过滤支持（SearchWithFilter）
- **需注意事项**：
  - ID 自动生成（int64），需转换为 string
  - 批量插入使用 InsertData 结构
  - 搜索结果包含 Metadata 字典

---

## 2. 项目约定

### 命名约定
- **包名**：小写，简短（biz, store, pool, metrics）
- **结构体**：驼峰命名（RAGService, MilvusStore, TreeBuilder）
- **接口**：功能命名 + er后缀（VectorStore, Indexer, Retriever, Summarizer）
- **配置结构体**：统一后缀 Config（ServiceConfig, IndexerConfig, TreeBuilderConfig）
- **私有方法**：小写开头（indexFiles, parseAndChunk, buildTree）

### 文件组织
- `internal/rag/biz/` - 业务逻辑层（service, indexer, retriever, generator, **tree_builder.go**, **tree_retriever.go**）
- `internal/rag/store/` - 存储抽象层（store.go 接口定义，milvus.go 实现）
- `pkg/infra/` - 基础设施公共库（pool, server, middleware）
- `configs/` - 配置文件（YAML格式）

### 导入顺序
1. 标准库（context, fmt, time, sync 等）
2. 第三方库（github.com/kart-io/logger, github.com/milvus-io/milvus/client/v2）
3. 项目内部库（github.com/kart-io/sentinel-x/internal/..., github.com/kart-io/sentinel-x/pkg/...）

### 代码风格
- **注释**：简体中文，描述意图和约束（不重复代码逻辑）
- **错误处理**：`fmt.Errorf("描述: %w", err)` 包装错误
- **日志记录**：
  - Info/Infof：正常流程关键步骤
  - Warn/Warnf：非致命错误（继续执行）
  - Error/Errorw：致命错误（返回失败）
- **defer**：用于资源清理、指标记录、panic恢复

---

## 3. 可复用组件清单

### 基础设施组件

#### 3.1 Goroutine Pool（pkg/infra/pool）
- **用途**：控制并发，避免goroutine泄漏
- **使用方式**：
  ```go
  import "github.com/kart-io/sentinel-x/pkg/infra/pool"

  // 提交到后台任务池（用于摘要生成）
  err := pool.SubmitToType(pool.BackgroundPool, func() {
      // 业务逻辑
  })

  // 降级处理（必须）
  if err != nil {
      logger.Warnw("池不可用，降级到goroutine", "error", err.Error())
      go func() {
          defer func() {
              if r := recover(); r != nil {
                  logger.Errorw("panic recovered", "error", r)
              }
          }()
          // 业务逻辑
      }()
  }
  ```
- **预定义池类型**：
  - `BackgroundPool`：后台任务（容量50，非阻塞，过期60s）- **用于树构建和摘要生成**
  - `DefaultPool`：通用任务（容量1000，阻塞）
  - `HealthCheckPool`：健康检查（容量100）
- **关键规范**：
  - 禁止直接使用 `go func()`
  - 必须提供降级处理
  - 池满时返回 ErrPoolOverload

#### 3.2 LLM Providers（pkg/llm）
- **ChatProvider 接口**：
  ```go
  type ChatProvider interface {
      Name() string
      Chat(ctx context.Context, messages []*ChatMessage) (*ChatResponse, error)
  }
  ```
- **用途**：生成树节点摘要
- **调用方式**：
  ```go
  messages := []*llm.ChatMessage{
      {Role: "user", Content: prompt},
  }
  resp, err := chatProvider.Chat(ctx, messages)
  if err != nil {
      return "", fmt.Errorf("摘要生成失败: %w", err)
  }
  summary := resp.Content
  ```

- **EmbeddingProvider 接口**：
  ```go
  type EmbeddingProvider interface {
      Name() string
      Embed(ctx context.Context, texts []string) ([][]float32, error)
      EmbedSingle(ctx context.Context, text string) ([]float32, error)
  }
  ```
- **用途**：生成摘要的向量
- **批量调用**：`embeddings, err := embedProvider.Embed(ctx, summaries)`

#### 3.3 文本处理工具（internal/pkg/rag/textutil）
- `SplitIntoChunks(text string, chunkSize, overlap int) []string` - 文本分块
- `SanitizeUTF8(text string) string` - 清理无效UTF-8字符
- `TruncateString(text string, maxLen int) string` - 截断字符串
- `HashString(text string) string` - 生成哈希ID

#### 3.4 统一日志库（github.com/kart-io/logger）
- **使用方式**：
  ```go
  import "github.com/kart-io/logger"

  logger.Infof("索引文档: %s", docName)
  logger.Warnw("警告信息", "key", value, "error", err.Error())
  logger.Errorw("错误信息", "key", value, "error", err.Error())
  ```

---

## 4. 测试策略

### 测试框架
- **框架**：Go testing + Testify（预计）
- **文件命名**：`*_test.go`（单元测试），`*_integration_test.go`（集成测试）

### Mock 策略
- **Mock VectorStore**：使用内存存储或测试桩
  ```go
  type mockVectorStore struct {
      chunks []*store.Chunk
  }

  func (m *mockVectorStore) Insert(ctx context.Context, collection string, chunks []*store.Chunk) ([]string, error) {
      m.chunks = append(m.chunks, chunks...)
      return generateIDs(len(chunks)), nil
  }
  ```

- **Mock ChatProvider**：返回固定摘要（避免真实LLM调用）
  ```go
  type mockChatProvider struct{}

  func (m *mockChatProvider) Chat(ctx context.Context, messages []*llm.ChatMessage) (*llm.ChatResponse, error) {
      return &llm.ChatResponse{
          Content: "这是一段测试摘要",
          TokenUsage: &llm.TokenUsage{PromptTokens: 10, CompletionTokens: 20},
      }, nil
  }
  ```

- **Mock EmbeddingProvider**：返回固定向量
  ```go
  func (m *mockEmbeddingProvider) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
      return make([]float32, 768), nil // 返回零向量
  }
  ```

### 测试覆盖标准
- **单元测试覆盖率**：> 80%
- **关键测试场景**：
  - TreeBuilder：小文档、大文档、递归层级、摘要生成失败
  - TreeRetriever：路径查找、叶子检索、综合排序
  - Schema扩展：字段正确性、索引创建、过滤查询

### 参考测试文件
- `internal/rag/biz/cache_test.go` - 缓存测试模式
- `internal/rag/metrics/metrics_test.go` - 指标测试模式
- `pkg/infra/pool/pool_test.go` - Pool 测试模式

---

## 5. 依赖和集成点

### 外部依赖
- **Milvus 向量数据库**：`github.com/milvus-io/milvus/client/v2`
  - 封装层：`pkg/component/milvus`
  - 配置：`configs/rag.yaml` 的 milvus 节点

- **LLM Providers**：
  - DeepSeek（默认）：用于摘要生成
  - Ollama：用于嵌入向量
  - 配置：`configs/rag.yaml` 的 chat 和 embedding 节点

- **Goroutine Pool**：`github.com/panjf2000/ants/v2`
  - 封装层：`pkg/infra/pool`

### 内部依赖
- **VectorStore 接口**：`internal/rag/store/store.go`
  - 需扩展：Chunk 结构体添加 Level, ParentID, NodeType 字段
  - 需新增：SearchWithFilter 方法（支持 Milvus 表达式过滤）

- **Indexer**：`internal/rag/biz/indexer.go`
  - 复用：parseAndChunk 方法获取叶子节点

- **Retriever**：`internal/rag/biz/retriever.go`
  - 参考：Retrieve 方法的结果结构和排序逻辑

### 集成方式
- **RAGService.IndexFromURL**：
  ```go
  func (s *RAGService) IndexFromURL(ctx context.Context, url string) error {
      // 现有逻辑
      err := s.indexer.IndexFromURL(ctx, url)

      // 新增：树索引（可选，失败不阻塞）
      if s.config.Tree.Enabled {
          if err := s.treeBuilder.BuildTree(ctx, url); err != nil {
              logger.Warnw("树索引构建失败，跳过", "error", err.Error())
          }
      }
      return err
  }
  ```

- **RAGService.Query**：
  ```go
  func (s *RAGService) Query(ctx context.Context, question string) (*model.QueryResult, error) {
      var results []*store.SearchResult

      // 向量检索（现有）
      vecResults, _ := s.retriever.Retrieve(ctx, question)

      // 树检索（新增，可选）
      if s.config.Tree.Enabled {
          treeResults, _ := s.treeRetriever.Retrieve(ctx, question)
          results = s.mergeResults(vecResults, treeResults)
      } else {
          results = vecResults.Results
      }

      // 生成答案
      return s.generator.GenerateAnswer(ctx, question, results)
  }
  ```

### 配置来源
- **配置文件**：`configs/rag.yaml`
- **扩展配置**（需添加）：
  ```yaml
  rag:
    # 现有配置...
    tree:
      enabled: false          # POC开关
      max-level: 3           # 树最大层级
      num-clusters: 5        # 每层聚类数
      summary-model: "deepseek-chat"  # 摘要模型
      summary-max-tokens: 200         # 摘要最大长度
  ```

---

## 6. 技术选型理由

### 为什么用树形索引（RAPTOR）
- **问题**：传统向量检索在复杂查询中容易遗漏高层语义
- **方案**：通过聚类和摘要构建多层树形索引，保留文档的层级语义
- **优势**：
  - 自顶向下检索能够捕捉全局主题
  - 叶子层检索保留细节信息
  - 混合检索兼顾准确性和召回率
- **劣势和风险**：
  - 索引构建时间较长（需要多次LLM调用）
  - 存储开销增加（约2倍）
  - 检索延迟可能增加（多次查询）

### 为什么用KMeans聚类
- **问题**：如何将叶子节点分组构建父节点
- **方案**：基于向量相似度的KMeans聚类
- **优势**：
  - 算法简单，易于实现
  - 可控的聚类数量（num_clusters=5）
  - 适合向量数据
- **备选方案**：
  - 层次聚类（复杂度高）
  - HDBSCAN（自动确定聚类数，但不可控）

### 为什么用Goroutine Pool
- **问题**：大量摘要生成任务需要并发控制
- **方案**：使用项目统一的 pool.BackgroundPool
- **优势**：
  - 避免goroutine泄漏
  - 控制并发数（容量50）
  - 统一的错误处理和降级
- **遵循项目规范**：CLAUDE.md强制要求

---

## 7. 关键风险点

### 并发问题
- **风险**：摘要生成大量并发可能导致池满或LLM限流
- **缓解**：
  - 使用 pool.BackgroundPool 控制并发（容量50）
  - 批量生成摘要（10个节点一批）
  - 添加重试机制和降级处理

### 边界条件
- **风险**：空节点、单节点、超大文档
- **缓解**：
  - 检查节点数量 ≤ 5 时停止递归
  - 设置 max_level=3 防止无限递归
  - 摘要失败时使用内容截断作为降级

### 性能瓶颈
- **风险**：树形检索多次查询导致延迟增加
- **缓解**：
  - 限制树深度（max_level=3）
  - 路径缓存（复用根节点查询结果）
  - 并行查询子节点
  - 超时降级到向量检索

### 存储开销
- **风险**：树节点数量 = 叶子数 × 层级因子
- **缓解**：
  - POC阶段监控实际开销
  - 实现TTL清理机制
  - 仅为重要文档构建树

### LLM摘要质量
- **风险**：摘要质量不稳定影响检索准确率
- **缓解**：
  - 使用模板化Prompt（提高稳定性）
  - 摘要长度限制（200 tokens）
  - 质量验证（摘要长度 > 20字符）
  - 失败降级（使用内容截断）

---

## 8. 实施优先级

### P0 - 基础设施（必须先完成）
1. 扩展 Chunk 结构体（添加 Level, ParentID, NodeType）
2. 扩展 Milvus Schema（MetaFields 添加字段）
3. 实现 SearchWithFilter 方法
4. 添加配置支持（rag.tree节点）

### P1 - 核心功能
1. 实现 TreeBuilder（tree_builder.go, cluster.go, summarizer.go）
2. 实现 TreeRetriever（tree_retriever.go, path_finder.go）
3. 集成到 RAGService

### P2 - 测试和验证
1. 单元测试（tree_builder_test.go, tree_retriever_test.go）
2. 集成测试（service_integration_test.go）
3. 对比测试（向量 vs 树 vs 融合）

---

## 9. 下一步行动

### 立即执行（阶段1：基础设施准备）
1. ✅ 生成本上下文摘要文件
2. ⏳ 查看 `pkg/component/milvus` 的 Client 实现
3. ⏳ 修改 `internal/rag/store/store.go` - 扩展 Chunk 结构体
4. ⏳ 修改 `internal/rag/store/milvus.go` - 扩展 Schema 和实现 SearchWithFilter
5. ⏳ 修改 `configs/rag.yaml` - 添加 tree 配置节点
6. ⏳ 运行单元测试验证基础设施

### 等待后续执行
- 阶段2：TreeBuilder 实现
- 阶段3：TreeRetriever 实现
- 阶段4：集成和验证

---

**摘要生成完毕** ✅
**上下文充分性**：已通过7项验证
**准备状态**：可以开始编码
