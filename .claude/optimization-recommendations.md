# Sentinel-X 项目优化建议报告

> **生成时间**: 2026-01-01
> **基于版本**: commit `ccdfe28`
> **分析深度**: 深度分析（结合项目分析文档、代码实现、配置文件）

---

## 目录

- [优先级说明](#优先级说明)
- [高优先级优化建议](#高优先级优化建议)
- [中优先级优化建议](#中优先级优化建议)
- [低优先级优化建议](#低优先级优化建议)
- [实施路线图](#实施路线图)

---

## 优先级说明

- **高优先级**: 影响系统稳定性、安全性或核心功能,需要立即处理
- **中优先级**: 影响性能、用户体验或开发效率,建议短期内处理
- **低优先级**: 优化改进项,可长期规划

---

## 高优先级优化建议

### H1. 配置管理安全增强

**问题描述**:
- 敏感信息（API Key）硬编码在配置文件中（`configs/rag.yaml` 中的 `${DEEPSEEK_API_KEY}`）
- 虽然使用了环境变量占位符,但配置文件中仍暴露了使用的 LLM 供应商信息
- 缺少统一的密钥管理机制

**优化方案**:
1. **引入配置管理工具**:
   - 使用 HashiCorp Vault 或 Kubernetes Secret 管理敏感信息
   - 开发环境使用 `.env` 文件（添加到 `.gitignore`）
   - 生产环境使用 Secret 管理服务

2. **配置分离**:
   ```yaml
   # configs/rag.yaml（公开配置）
   chat:
     provider: "deepseek"
     base-url: "https://api.deepseek.com"
     model: "deepseek-chat"
     # API Key 从环境变量或 Secret 加载
   ```

3. **实现配置加载优先级**:
   - 环境变量 > Secret 文件 > 配置文件默认值
   - 参考实现: `pkg/infra/config/loader.go` 中增加 Secret 加载逻辑

**预期收益**:
- ✅ 消除敏感信息泄露风险
- ✅ 符合安全合规要求
- ✅ 支持多环境配置（开发、测试、生产）

**实施难度**: ⭐⭐（中等）
**实施时间**: 1-2 天
**相关文件**:
- `pkg/infra/config/config.go`
- `configs/rag.yaml`
- `configs/user-center.yaml`

---

### H2. RAG 查询结果缓存层

**问题描述**:
- 当前每次 RAG 查询都会调用 LLM（DeepSeek）,响应时间长（60s 超时）
- 相同问题重复查询会产生重复的 LLM 调用成本（API 费用）
- 文档第九章提到"引入缓存层（Redis）缓存 RAG 查询结果"

**优化方案**:
1. **实现查询结果缓存**:
   ```go
   // internal/rag/biz/cache.go
   type CachedGenerator struct {
       generator Generator
       cache     cache.Cache // 复用 pkg/cache 接口
       ttl       time.Duration
   }

   func (g *CachedGenerator) Generate(ctx context.Context, question string, contexts []Source) (*Result, error) {
       // 1. 生成缓存键（基于问题哈希）
       cacheKey := generateCacheKey(question)

       // 2. 尝试从缓存获取
       if cached, err := g.cache.Get(ctx, cacheKey); err == nil {
           return parseCachedResult(cached), nil
       }

       // 3. 缓存未命中，调用 LLM
       result, err := g.generator.Generate(ctx, question, contexts)
       if err != nil {
           return nil, err
       }

       // 4. 写入缓存
       g.cache.Set(ctx, cacheKey, serializeResult(result), g.ttl)
       return result, nil
   }
   ```

2. **配置缓存策略**:
   ```yaml
   # configs/rag.yaml
   rag:
     cache:
       enabled: true
       ttl: 3600  # 缓存 1 小时
       strategy: question-hash  # 基于问题哈希
       max-size: 1000  # 最大缓存条目数
   ```

3. **缓存失效策略**:
   - 时间失效（TTL）
   - 手动清除（当知识库更新时）
   - LRU 淘汰（基于 Redis）

**预期收益**:
- ✅ 减少 80% 的重复 LLM 调用成本（假设 20% 问题重复）
- ✅ 响应时间从 60s 降低到 <100ms（缓存命中）
- ✅ 降低 DeepSeek API 费用

**实施难度**: ⭐⭐⭐（中等）
**实施时间**: 2-3 天
**相关文件**:
- `internal/rag/biz/generator.go`
- `pkg/cache/cache.go`（已存在缓存接口）
- `pkg/component/redis/redis.go`

---

### H3. 清理废弃代码和架构遗留

**问题描述**:
- `internal/bootstrap/` 目录已废弃但未删除（文档第九章明确指出）
- 最近提交中有 `refact: del auth` 和 `refactor: 移除Factory模式和bootstrap依赖`,表明架构重构过程中有反复
- 废弃代码增加维护成本和理解难度

**优化方案**:
1. **删除废弃目录**:
   ```bash
   rm -rf internal/bootstrap/
   ```

2. **检查引用**:
   ```bash
   # 确保没有代码引用 bootstrap 包
   grep -r "internal/bootstrap" --include="*.go" .
   ```

3. **更新文档**:
   - 在 `docs/design/architecture.md` 中移除 bootstrap 层的说明
   - 更新 README 中的项目结构说明

4. **清理其他废弃代码**:
   - 检查是否有其他未使用的包或文件
   - 使用工具检测无效代码: `deadcode` 或 `golangci-lint`

**预期收益**:
- ✅ 减少代码库体积
- ✅ 降低新开发者理解成本
- ✅ 避免误用废弃代码

**实施难度**: ⭐（简单）
**实施时间**: 1 小时
**相关文件**:
- `internal/bootstrap/`（删除）
- `docs/design/architecture.md`（更新）

---

### H4. 测试覆盖率提升与可视化

**问题描述**:
- 虽然有 `coverage.out` 文件,但具体覆盖率未公开
- 部分核心模块（RAG）缺少集成测试
- 缺少自动化测试报告

**优化方案**:
1. **生成测试覆盖率报告**:
   ```makefile
   # Makefile 增加目标
   .PHONY: test-coverage-report
   test-coverage-report:
       @echo "==> Generating test coverage report"
       @go test -coverprofile=coverage.out ./...
       @go tool cover -html=coverage.out -o coverage.html
       @go tool cover -func=coverage.out | grep total
   ```

2. **设定覆盖率目标**:
   - 核心业务逻辑（`internal/*/biz/`）: ≥ 80%
   - 数据访问层（`internal/*/store/`）: ≥ 70%
   - 工具函数（`pkg/utils/`）: ≥ 90%

3. **增加 RAG 集成测试**:
   ```go
   // internal/rag/biz/service_test.go
   func TestRAGService_Integration(t *testing.T) {
       // 使用 testcontainers 启动 Milvus
       // 模拟完整的索引和查询流程
   }
   ```

4. **CI 集成**:
   ```yaml
   # .github/workflows/test.yml
   - name: Test with coverage
     run: make test-coverage-report

   - name: Upload coverage to Codecov
     uses: codecov/codecov-action@v3
   ```

**预期收益**:
- ✅ 可视化测试覆盖率
- ✅ 发现未测试的关键路径
- ✅ 提升代码质量信心

**实施难度**: ⭐⭐⭐（中等）
**实施时间**: 3-5 天
**相关文件**:
- `Makefile`
- `.github/workflows/test.yml`
- `internal/rag/biz/service_test.go`（新增）

---

## 中优先级优化建议

### M1. Milvus 分片策略优化

**问题描述**:
- 当前使用单一集合（`milvus_docs`）存储所有文档
- 文档第九章提到"Milvus 分片策略优化（大规模数据）"
- 随着知识库增长,单一集合可能成为性能瓶颈

**优化方案**:
1. **按文档来源分片**:
   ```go
   // 不同来源使用不同集合
   collections := map[string]string{
       "official_docs": "milvus_official_docs",
       "github_issues": "milvus_github_issues",
       "blog_posts":    "milvus_blog_posts",
   }
   ```

2. **实现集合路由**:
   ```go
   // internal/rag/biz/router.go
   type CollectionRouter struct {
       collections map[string]string
   }

   func (r *CollectionRouter) Route(source string) string {
       if coll, ok := r.collections[source]; ok {
           return coll
       }
       return "default_collection"
   }
   ```

3. **配置分片策略**:
   ```yaml
   # configs/rag.yaml
   rag:
     collections:
       - name: "milvus_official_docs"
         source-pattern: "https://milvus.io/docs/**"
         shard-num: 2
       - name: "milvus_github_issues"
         source-pattern: "https://github.com/milvus-io/**"
         shard-num: 4
   ```

**预期收益**:
- ✅ 支持大规模数据（百万级文档）
- ✅ 查询性能提升（减少搜索范围）
- ✅ 便于管理和维护（按来源清理数据）

**实施难度**: ⭐⭐⭐⭐（较高）
**实施时间**: 5-7 天
**相关文件**:
- `internal/rag/biz/service.go`
- `pkg/component/milvus/milvus.go`
- `configs/rag.yaml`

---

### M2. LLM 调用重试与熔断机制

**问题描述**:
- 文档第九章提到"LLM 调用增加重试和超时熔断"
- 当前配置中有 `max-retries: 3`,但实现中可能缺少熔断机制
- LLM 服务不稳定时,可能导致大量请求堆积

**优化方案**:
1. **实现重试机制**:
   ```go
   // pkg/llm/retry.go
   type RetryableProvider struct {
       provider    ChatProvider
       maxRetries  int
       backoff     time.Duration
   }

   func (p *RetryableProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
       var lastErr error
       for i := 0; i < p.maxRetries; i++ {
           resp, err := p.provider.Chat(ctx, req)
           if err == nil {
               return resp, nil
           }
           lastErr = err

           // 指数退避
           time.Sleep(p.backoff * time.Duration(1<<i))
       }
       return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
   }
   ```

2. **集成熔断器**:
   ```go
   // 使用 sony/gobreaker
   import "github.com/sony/gobreaker"

   type CircuitBreakerProvider struct {
       provider ChatProvider
       breaker  *gobreaker.CircuitBreaker
   }

   func (p *CircuitBreakerProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
       result, err := p.breaker.Execute(func() (interface{}, error) {
           return p.provider.Chat(ctx, req)
       })
       if err != nil {
           return nil, err
       }
       return result.(*ChatResponse), nil
   }
   ```

3. **配置策略**:
   ```yaml
   # configs/rag.yaml
   chat:
     retry:
       max-retries: 3
       backoff: 1s
     circuit-breaker:
       max-requests: 3  # 熔断前最大失败次数
       interval: 10s    # 统计周期
       timeout: 60s     # 熔断后等待时间
   ```

**预期收益**:
- ✅ 提升系统稳定性（应对 LLM 服务波动）
- ✅ 避免雪崩效应
- ✅ 改善用户体验（重试自动恢复）

**实施难度**: ⭐⭐⭐（中等）
**实施时间**: 2-3 天
**相关文件**:
- `pkg/llm/provider.go`
- `pkg/llm/retry.go`（新增）
- `pkg/llm/circuit_breaker.go`（新增）

---

### M3. 监控与告警体系建设

**问题描述**:
- 文档第九章提到"集成 Grafana Dashboard（可视化指标）"和"配置告警规则（Prometheus Alertmanager）"
- 当前已有 Prometheus 指标导出（`/metrics` 端点）,但缺少可视化和告警
- 无法及时发现系统异常（如 RAG 查询失败率上升、LLM 调用超时）

**优化方案**:
1. **Grafana Dashboard 开发**:
   ```json
   // deploy/grafana/dashboards/sentinel-rag.json
   {
     "title": "Sentinel RAG Service",
     "panels": [
       {
         "title": "RAG Query QPS",
         "targets": [
           {
             "expr": "rate(sentinel_rag_query_total[5m])"
           }
         ]
       },
       {
         "title": "RAG Query Latency (P95)",
         "targets": [
           {
             "expr": "histogram_quantile(0.95, sentinel_rag_query_duration_seconds)"
           }
         ]
       },
       {
         "title": "LLM Call Success Rate",
         "targets": [
           {
             "expr": "rate(sentinel_rag_llm_success_total[5m]) / rate(sentinel_rag_llm_total[5m])"
           }
         ]
       }
     ]
   }
   ```

2. **Prometheus 告警规则**:
   ```yaml
   # deploy/prometheus/alerts/sentinel-rag.yml
   groups:
     - name: sentinel-rag
       interval: 30s
       rules:
         - alert: HighRAGQueryFailureRate
           expr: rate(sentinel_rag_query_errors_total[5m]) / rate(sentinel_rag_query_total[5m]) > 0.1
           for: 5m
           labels:
             severity: warning
           annotations:
             summary: "RAG 查询失败率过高（> 10%）"
             description: "当前失败率: {{ $value }}"

         - alert: LLMCallTimeout
           expr: rate(sentinel_rag_llm_timeout_total[5m]) > 5
           for: 2m
           labels:
             severity: critical
           annotations:
             summary: "LLM 调用频繁超时"
   ```

3. **增加业务指标**:
   ```go
   // internal/rag/biz/metrics.go
   var (
       queryTotal = prometheus.NewCounterVec(
           prometheus.CounterOpts{
               Name: "sentinel_rag_query_total",
               Help: "Total number of RAG queries",
           },
           []string{"status"},
       )

       queryDuration = prometheus.NewHistogramVec(
           prometheus.HistogramOpts{
               Name: "sentinel_rag_query_duration_seconds",
               Help: "RAG query duration in seconds",
               Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60},
           },
           []string{"operation"},
       )
   )
   ```

**预期收益**:
- ✅ 实时监控系统运行状态
- ✅ 快速发现和定位问题
- ✅ 告警自动通知（邮件、钉钉、Slack）

**实施难度**: ⭐⭐⭐（中等）
**实施时间**: 3-4 天
**相关文件**:
- `deploy/grafana/dashboards/`（新增）
- `deploy/prometheus/alerts/`（新增）
- `internal/rag/biz/metrics.go`（新增）

---

### M4. 向量检索参数自动调优

**问题描述**:
- 当前 HNSW 索引参数硬编码（`M=16, efConstruction=200, ef=64`）
- 不同数据规模和查询场景下,最优参数可能不同
- 缺少性能基准测试

**优化方案**:
1. **实现参数配置化**:
   ```yaml
   # configs/rag.yaml
   milvus:
     index:
       type: HNSW
       params:
         M: 16              # 每个节点的连接数
         efConstruction: 200  # 构建质量
       search:
         ef: 64             # 搜索质量
   ```

2. **开发性能基准测试**:
   ```go
   // internal/rag/benchmark/search_bench_test.go
   func BenchmarkSearch_HNSW_M16(b *testing.B) {
       // 测试不同参数组合的性能
   }

   func BenchmarkSearch_HNSW_M32(b *testing.B) {
       // ...
   }
   ```

3. **自动调优脚本**:
   ```bash
   # scripts/benchmark/tune-hnsw.sh
   for M in 8 16 32; do
       for ef in 32 64 128; do
           echo "Testing M=$M, ef=$ef"
           # 运行基准测试并记录结果
       done
   done
   ```

**预期收益**:
- ✅ 提升检索性能（找到最优参数）
- ✅ 平衡速度与准确率
- ✅ 支持动态调整（适应数据增长）

**实施难度**: ⭐⭐⭐⭐（较高）
**实施时间**: 4-5 天
**相关文件**:
- `pkg/component/milvus/milvus.go`
- `configs/rag.yaml`
- `scripts/benchmark/tune-hnsw.sh`（新增）

---

### M5. 文档同步与一致性保障

**问题描述**:
- 文档第九章提到"部分文档（如 `docs/design/architecture.md`）未更新 RAG 服务"
- Swagger 文档未自动同步到代码
- 架构图与实际实现可能不一致

**优化方案**:
1. **架构文档更新**:
   - 在 `docs/design/architecture.md` 中补充 RAG 服务的架构说明
   - 绘制最新的服务架构图（包含 RAG Service）
   - 更新数据流图（文档索引流程、查询流程）

2. **Swagger 自动同步**:
   ```go
   // 使用 swaggo/swag 自动生成
   // cmd/rag/main.go

   // @title Sentinel RAG API
   // @version 1.0
   // @description RAG 知识库服务 API
   // @host localhost:8082
   // @BasePath /api/v1

   //go:generate swag init -g main.go -o ../../api/swagger
   ```

3. **文档一致性检查**:
   ```bash
   # scripts/check-docs.sh
   # 检查文档中提到的文件路径是否存在
   # 检查架构图与代码结构是否匹配
   ```

**预期收益**:
- ✅ 降低新开发者理解成本
- ✅ 保持文档与代码同步
- ✅ 自动化文档生成

**实施难度**: ⭐⭐（中等）
**实施时间**: 2-3 天
**相关文件**:
- `docs/design/architecture.md`
- `cmd/rag/main.go`
- `scripts/check-docs.sh`（新增）

---

## 低优先级优化建议

### L1. CI/CD 自动部署流水线

**问题描述**:
- 文档第九章提到"`.github/workflows/` 已有工作流,但未启用自动部署"
- 当前需要手动构建和部署
- 缺少自动化测试和部署流程

**优化方案**:
1. **完善 CI 流水线**:
   ```yaml
   # .github/workflows/ci.yml
   name: CI
   on:
     push:
       branches: [master, develop]
     pull_request:
       branches: [master]

   jobs:
     test:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v4
         - uses: actions/setup-go@v5
           with:
             go-version: '1.25.0'
         - run: make test-coverage-report
         - run: make lint

     build:
       runs-on: ubuntu-latest
       needs: test
       steps:
         - run: make build
         - uses: docker/build-push-action@v5
           with:
             push: true
             tags: sentinel-x/rag:${{ github.sha }}
   ```

2. **实现 CD 流水线**:
   ```yaml
   # .github/workflows/cd.yml
   name: CD
   on:
     push:
       tags:
         - 'v*'

   jobs:
     deploy:
       runs-on: ubuntu-latest
       steps:
         - name: Deploy to Kubernetes
           run: |
             kubectl apply -f deploy/k8s/rag-deployment.yaml
             kubectl set image deployment/rag rag=sentinel-x/rag:${{ github.ref_name }}
   ```

**预期收益**:
- ✅ 自动化构建和测试
- ✅ 快速部署（分钟级）
- ✅ 减少人工错误

**实施难度**: ⭐⭐⭐（中等）
**实施时间**: 3-4 天
**相关文件**:
- `.github/workflows/ci.yml`
- `.github/workflows/cd.yml`
- `deploy/k8s/`（新增）

---

### L2. HTTPS/TLS 支持

**问题描述**:
- 当前仅支持 HTTP
- 生产环境需要 HTTPS 加密传输
- 敏感信息（JWT Token、API Key）可能被中间人攻击

**优化方案**:
1. **配置 TLS**:
   ```yaml
   # configs/rag.yaml
   server:
     http:
       tls:
         enabled: true
         cert-file: /etc/sentinel/tls/server.crt
         key-file: /etc/sentinel/tls/server.key
   ```

2. **实现 TLS 加载**:
   ```go
   // pkg/infra/server/http/server.go
   if opts.TLS.Enabled {
       srv.ListenAndServeTLS(opts.TLS.CertFile, opts.TLS.KeyFile)
   } else {
       srv.ListenAndServe()
   }
   ```

3. **自动证书管理**:
   - 使用 cert-manager（Kubernetes）
   - 使用 Let's Encrypt（自动续期）

**预期收益**:
- ✅ 传输加密（防止中间人攻击）
- ✅ 符合安全合规要求
- ✅ 支持 HTTPS 证书验证

**实施难度**: ⭐⭐（中等）
**实施时间**: 1-2 天
**相关文件**:
- `pkg/infra/server/http/server.go`
- `pkg/options/server/options.go`
- `configs/*.yaml`

---

### L3. 查询增强策略智能选择

**问题描述**:
- 当前查询增强功能（Query Rewriting、HyDE、Reranking）全局开关
- 配置文件中全部禁用（`enable-query-rewrite: false`）以节省时间
- 实际场景中,不同类型问题可能需要不同的增强策略

**优化方案**:
1. **实现策略选择器**:
   ```go
   // internal/pkg/rag/enhancer/strategy.go
   type StrategySelector struct {
       rules []Rule
   }

   type Rule struct {
       Condition func(question string) bool
       Enhancers []string // ["rewrite", "hyde", "rerank"]
   }

   func (s *StrategySelector) Select(question string) []string {
       for _, rule := range s.rules {
           if rule.Condition(question) {
               return rule.Enhancers
           }
       }
       return []string{} // 默认不增强
   }
   ```

2. **配置规则**:
   ```yaml
   # configs/rag.yaml
   rag:
     enhancer:
       auto-select: true
       rules:
         - name: "复杂问题"
           condition: "length > 50 || contains('如何', '为什么')"
           enhancers: ["rewrite", "rerank"]
         - name: "简单查询"
           condition: "length < 20"
           enhancers: []
   ```

3. **基于历史数据优化**:
   - 记录不同策略的效果（忠实度、相关性评分）
   - 使用机器学习模型选择最优策略

**预期收益**:
- ✅ 平衡速度与质量
- ✅ 降低不必要的增强开销
- ✅ 提升用户体验（快速响应 + 高质量答案）

**实施难度**: ⭐⭐⭐⭐（较高）
**实施时间**: 5-7 天
**相关文件**:
- `internal/pkg/rag/enhancer/strategy.go`（新增）
- `internal/rag/biz/retriever.go`
- `configs/rag.yaml`

---

### L4. 多租户支持

**问题描述**:
- 当前架构假设单租户使用
- 多个团队使用同一知识库时缺少隔离
- 无法按租户进行资源配额和计费

**优化方案**:
1. **集合级隔离**:
   ```go
   // internal/rag/biz/tenant.go
   type TenantManager struct {
       collectionPrefix string
   }

   func (m *TenantManager) GetCollection(tenantID string) string {
       return fmt.Sprintf("%s_%s", m.collectionPrefix, tenantID)
   }
   ```

2. **租户识别**:
   ```go
   // pkg/infra/middleware/tenant/tenant.go
   func TenantMiddleware() gin.HandlerFunc {
       return func(c *gin.Context) {
           tenantID := c.GetHeader("X-Tenant-ID")
           if tenantID == "" {
               c.AbortWithStatus(http.StatusBadRequest)
               return
           }
           c.Set("tenant_id", tenantID)
           c.Next()
       }
   }
   ```

3. **资源配额**:
   ```yaml
   # configs/rag.yaml
   tenants:
     - id: "team-a"
       quota:
         max-documents: 10000
         max-queries-per-day: 1000
     - id: "team-b"
       quota:
         max-documents: 50000
         max-queries-per-day: 5000
   ```

**预期收益**:
- ✅ 支持多团队使用
- ✅ 数据隔离（安全性）
- ✅ 资源配额控制

**实施难度**: ⭐⭐⭐⭐⭐（高）
**实施时间**: 10-14 天
**相关文件**:
- `internal/rag/biz/tenant.go`（新增）
- `pkg/infra/middleware/tenant/`（新增）
- `configs/rag.yaml`

---

### L5. 查询日志与审计

**问题描述**:
- 缺少详细的查询日志（问题、答案、来源、耗时）
- 无法分析用户查询模式
- 缺少审计功能（谁在什么时候查询了什么）

**优化方案**:
1. **实现查询日志记录**:
   ```go
   // internal/rag/biz/audit.go
   type QueryLog struct {
       ID          string    `json:"id"`
       TenantID    string    `json:"tenant_id"`
       UserID      string    `json:"user_id"`
       Question    string    `json:"question"`
       Answer      string    `json:"answer"`
       Sources     []string  `json:"sources"`
       Duration    int64     `json:"duration_ms"`
       Timestamp   time.Time `json:"timestamp"`
   }

   func (s *Service) logQuery(ctx context.Context, log *QueryLog) error {
       // 写入数据库或日志文件
       return s.auditStore.Save(ctx, log)
   }
   ```

2. **查询分析**:
   ```sql
   -- 最常见的问题
   SELECT question, COUNT(*) as count
   FROM query_logs
   GROUP BY question
   ORDER BY count DESC
   LIMIT 10;

   -- 慢查询
   SELECT question, duration_ms
   FROM query_logs
   WHERE duration_ms > 10000
   ORDER BY duration_ms DESC;
   ```

3. **审计报告**:
   - 定期生成审计报告（每日/每周）
   - 发送给管理员

**预期收益**:
- ✅ 了解用户查询模式
- ✅ 发现系统瓶颈
- ✅ 满足合规要求（审计）

**实施难度**: ⭐⭐⭐（中等）
**实施时间**: 3-4 天
**相关文件**:
- `internal/rag/biz/audit.go`（新增）
- `internal/rag/store/audit_store.go`（新增）

---

## 实施路线图

### 第一阶段（1-2 周）：基础优化 - 稳定性与安全性

**目标**: 解决高优先级问题,提升系统稳定性和安全性

| 任务 | 优先级 | 实施时间 | 责任人 |
|------|--------|----------|--------|
| H3. 清理废弃代码 | 高 | 1 小时 | 开发 |
| H1. 配置管理安全增强 | 高 | 1-2 天 | 开发 + 运维 |
| H4. 测试覆盖率提升 | 高 | 3-5 天 | QA + 开发 |
| H2. RAG 查询结果缓存 | 高 | 2-3 天 | 开发 |

**验收标准**:
- ✅ 废弃代码已删除,文档已更新
- ✅ 敏感信息使用 Secret 管理,配置文件中无明文密钥
- ✅ 测试覆盖率报告可视化,核心模块覆盖率 ≥ 80%
- ✅ RAG 查询缓存命中率 ≥ 20%,响应时间降低 50%

---

### 第二阶段（2-3 周）：性能优化 - 吞吐量与响应时间

**目标**: 提升系统性能,优化用户体验

| 任务 | 优先级 | 实施时间 | 责任人 |
|------|--------|----------|--------|
| M2. LLM 调用重试与熔断 | 中 | 2-3 天 | 开发 |
| M4. 向量检索参数自动调优 | 中 | 4-5 天 | 开发 + 算法 |
| M1. Milvus 分片策略优化 | 中 | 5-7 天 | 开发 + DBA |

**验收标准**:
- ✅ LLM 调用失败率 < 1%（重试生效）
- ✅ 熔断器在 LLM 服务故障时正常工作
- ✅ 向量检索 P95 延迟 < 100ms
- ✅ 支持 100 万级文档规模

---

### 第三阶段（1-2 周）：运维增强 - 可观测性与自动化

**目标**: 完善监控告警,实现自动化运维

| 任务 | 优先级 | 实施时间 | 责任人 |
|------|--------|----------|--------|
| M3. 监控与告警体系 | 中 | 3-4 天 | 运维 + 开发 |
| M5. 文档同步与一致性 | 中 | 2-3 天 | 技术写作 + 开发 |
| L1. CI/CD 自动部署 | 低 | 3-4 天 | 运维 |

**验收标准**:
- ✅ Grafana Dashboard 完成,展示核心指标
- ✅ Prometheus 告警规则生效,测试告警通知
- ✅ 文档与代码同步,架构图更新
- ✅ CI/CD 流水线运行,自动测试和部署

---

### 第四阶段（2-3 周）：功能增强 - 高级特性

**目标**: 实现高级功能,提升产品竞争力

| 任务 | 优先级 | 实施时间 | 责任人 |
|------|--------|----------|--------|
| L2. HTTPS/TLS 支持 | 低 | 1-2 天 | 运维 + 开发 |
| L3. 查询增强策略智能选择 | 低 | 5-7 天 | 算法 + 开发 |
| L5. 查询日志与审计 | 低 | 3-4 天 | 开发 |
| L4. 多租户支持 | 低 | 10-14 天 | 架构师 + 开发 |

**验收标准**:
- ✅ HTTPS 启用,证书自动续期
- ✅ 查询增强策略自动选择,响应时间优化 30%
- ✅ 查询日志记录完整,支持审计查询
- ✅ 多租户隔离生效,支持配额管理

---

## 总结

### 优化收益预估

| 维度 | 当前状态 | 优化后目标 | 提升幅度 |
|------|----------|------------|----------|
| **响应时间** | 60s（RAG 查询） | 5s（缓存命中） | **↓ 92%** |
| **系统稳定性** | 未知（无监控） | 99.9% 可用性 | **显著提升** |
| **安全性** | 中等（敏感信息硬编码） | 高（Secret 管理） | **↑ 40%** |
| **测试覆盖率** | 未知 | 核心模块 ≥ 80% | **可量化** |
| **运维效率** | 手动部署 | 自动化 CI/CD | **↑ 70%** |
| **成本优化** | 无缓存（重复调用） | 缓存 + 熔断 | **↓ 30% LLM 费用** |

### 关键成功因素

1. **分阶段实施**: 按优先级逐步推进,避免过度设计
2. **持续验证**: 每个阶段都有明确的验收标准
3. **文档同步**: 代码和文档保持一致,降低理解成本
4. **自动化优先**: 测试、部署、监控全面自动化
5. **性能基准**: 建立性能基准,量化优化效果

### 风险提示

⚠️ **注意事项**:
- 多租户支持（L4）改动较大,建议充分测试后再上线
- Milvus 分片策略优化（M1）需要数据迁移,提前备份
- 查询增强策略智能选择（L3）涉及算法调优,需要足够的历史数据

---

**报告完成时间**: 2026-01-01
**下一步行动**: 根据实施路线图启动第一阶段优化工作
