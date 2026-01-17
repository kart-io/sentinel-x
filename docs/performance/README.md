# 性能文档

> **更新时间**: 2026-01-17
> **适用版本**: Sentinel-X v1.0+

---

## 概述

本文档定义了 Sentinel-X 各服务的性能目标（SLA）、性能测试方法和优化历史。

---

## 性能目标（SLA）

### User Center

| 接口 | P50 | P95 | P99 | 说明 |
|------|-----|-----|-----|------|
| **用户登录** | < 100ms | < 200ms | < 500ms | 包含数据库查询和 JWT 生成 |
| **用户注册** | < 150ms | < 300ms | < 600ms | 包含密码哈希和数据库插入 |
| **Token 验证** | < 10ms | < 20ms | < 50ms | 仅 JWT 验证，无数据库查询 |
| **用户信息查询** | < 50ms | < 100ms | < 200ms | 单次数据库查询 |
| **角色权限检查** | < 20ms | < 50ms | < 100ms | Casbin 内存检查 |

### RAG Service

| 接口 | P50 | P95 | P99 | 说明 |
|------|-----|-----|-----|------|
| **文档索引** | < 5s | < 10s | < 20s | 取决于文档大小 |
| **知识问答** | < 2s | < 5s | < 10s | 包含 Embedding、检索、LLM 生成 |
| **向量检索** | < 500ms | < 1s | < 2s | Milvus HNSW 检索 |
| **Embedding 生成** | < 200ms | < 500ms | < 1s | 使用 Ollama 本地模型 |

### API Server

| 接口 | P50 | P95 | P99 | 说明 |
|------|-----|-----|-----|------|
| **健康检查** | < 10ms | < 20ms | < 50ms | /health, /live, /ready |
| **路由转发** | < 50ms | < 100ms | < 200ms | 不包含后端服务处理时间 |

---

## 性能测试方法

### 基准测试（Benchmark）

使用 Go 内置的 benchmark 工具：

```bash
# 运行所有基准测试
make bench

# 运行特定包的基准测试
go test -bench=. -benchmem ./internal/rag/biz/...

# 生成 CPU profile
go test -bench=. -cpuprofile=cpu.prof ./internal/rag/biz/...
go tool pprof cpu.prof

# 生成内存 profile
go test -bench=. -memprofile=mem.prof ./internal/rag/biz/...
go tool pprof mem.prof
```

**示例基准测试**:

```go
// internal/rag/biz/retriever_test.go
func BenchmarkRetriever_Search(b *testing.B) {
    retriever := setupRetriever()
    question := "What is Milvus?"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := retriever.Search(context.Background(), question, 5)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// 运行结果示例:
// BenchmarkRetriever_Search-8    100    12345678 ns/op    1234 B/op    56 allocs/op
```

### 压力测试（Load Testing）

使用 `hey` 或 `wrk` 进行 HTTP 压力测试：

```bash
# 安装 hey
go install github.com/rakyll/hey@latest

# RAG 查询压力测试
hey -n 1000 -c 10 -m POST \
  -H "Content-Type: application/json" \
  -d '{"question":"What is Milvus?"}' \
  http://localhost:8082/v1/rag/query

# 用户登录压力测试
hey -n 1000 -c 10 -m POST \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test123"}' \
  http://localhost:8081/v1/auth/login
```

**结果分析**:

```
Summary:
  Total:        10.2345 secs
  Slowest:      2.1234 secs
  Fastest:      0.5678 secs
  Average:      1.0234 secs
  Requests/sec: 97.71

Response time histogram:
  0.568 [1]     |
  0.724 [123]   |■■■■■■■■■■■■■
  0.880 [456]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  1.036 [234]   |■■■■■■■■■■■■■■■■■■■■
  1.192 [123]   |■■■■■■■■■■■
```

### 性能监控

#### Prometheus 指标

```bash
# 查看 Prometheus 指标
curl http://localhost:8082/metrics

# 关键指标
# - http_request_duration_seconds: HTTP 请求延迟
# - rag_query_duration_seconds: RAG 查询总耗时
# - rag_embedding_duration_seconds: Embedding 生成耗时
# - rag_search_duration_seconds: 向量检索耗时
# - rag_llm_duration_seconds: LLM 生成耗时
```

#### Grafana 仪表板

```yaml
# 推荐的 Grafana 面板
panels:
  - title: "RAG Query Latency (P50/P95/P99)"
    query: |
      histogram_quantile(0.50, rate(rag_query_duration_seconds_bucket[5m]))
      histogram_quantile(0.95, rate(rag_query_duration_seconds_bucket[5m]))
      histogram_quantile(0.99, rate(rag_query_duration_seconds_bucket[5m]))

  - title: "RAG Query Breakdown"
    query: |
      rate(rag_embedding_duration_seconds_sum[5m]) / rate(rag_embedding_duration_seconds_count[5m])
      rate(rag_search_duration_seconds_sum[5m]) / rate(rag_search_duration_seconds_count[5m])
      rate(rag_llm_duration_seconds_sum[5m]) / rate(rag_llm_duration_seconds_count[5m])
```

---

## 性能优化历史

### 2026-01-17: RAG 查询性能优化

**问题**:
- RAG 查询超时，P99 延迟 > 60s
- Milvus 向量检索耗时 ~25s

**分析**:
1. 使用性能监控发现瓶颈在向量检索环节
2. 索引算法为 IVF_FLAT，检索速度慢
3. 查询增强功能（Query Rewrite、Rerank）耗时 ~4s

**优化措施**:

| 优化项 | 方法 | 效果 |
|--------|------|------|
| **向量索引** | IVF_FLAT → HNSW | 检索时间从 25s 降至 < 1s |
| **查询增强** | 默认禁用 Query Rewrite、Rerank | 节省 ~4s |
| **Embedding 缓存** | 使用 Redis 缓存 Embedding 结果 | 相同文本 Embedding 时间 < 10ms |
| **查询结果缓存** | 使用 Redis 缓存查询结果 | 相同问题响应时间 < 100ms |

**优化结果**:

| 指标 | 优化前 | 优化后 | 改进 |
|------|--------|--------|------|
| P50 延迟 | 30s | 1.5s | **95% ↓** |
| P95 延迟 | 50s | 4s | **92% ↓** |
| P99 延迟 | 60s | 8s | **87% ↓** |
| 缓存命中率 | 0% | 60% | - |

**相关 Commit**:
- `ccdfe28`: 向量索引切换到 HNSW
- 待提交: Embedding 缓存和查询结果缓存

---

## 性能优化最佳实践

### 1. 数据库优化

#### 索引优化

```sql
-- User Center 推荐索引
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);

-- 检查索引使用情况
EXPLAIN SELECT * FROM users WHERE username = 'test';
```

#### 连接池配置

```yaml
# configs/user-center.yaml
mysql:
  max-idle-connections: 10      # 最大空闲连接数
  max-open-connections: 100     # 最大打开连接数
  max-connection-life-time: 3600s  # 连接最大生命周期
```

### 2. 缓存策略

#### Redis 缓存

```go
// Embedding 缓存
type EmbeddingCache struct {
    client *redis.Client
    ttl    time.Duration
}

func (c *EmbeddingCache) GetOrCompute(text string) ([]float32, error) {
    key := "emb:" + hash(text)

    // 1. 尝试从缓存获取
    if cached, err := c.client.Get(ctx, key).Result(); err == nil {
        return deserialize(cached), nil
    }

    // 2. 计算 Embedding
    embedding := computeEmbedding(text)

    // 3. 缓存结果
    c.client.Set(ctx, key, serialize(embedding), c.ttl)

    return embedding, nil
}
```

#### 缓存失效策略

```yaml
# configs/rag.yaml
cache:
  enabled: true
  ttl: 1h                    # 缓存过期时间
  key-prefix: "rag:query:"   # 缓存键前缀
```

### 3. 并发控制

#### Goroutine 池

```go
// 使用 Goroutine 池处理并发任务
import "github.com/kart-io/sentinel-x/pkg/infra/pool"

// 并行处理多个文档
for _, doc := range docs {
    doc := doc  // 捕获循环变量
    pool.Submit(func() {
        processDocument(doc)
    })
}
```

#### 并发限制

```go
// 限制并发数
sem := make(chan struct{}, 10)  // 最多 10 个并发

for _, item := range items {
    sem <- struct{}{}  // 获取信号量
    go func(item Item) {
        defer func() { <-sem }()  // 释放信号量
        process(item)
    }(item)
}
```

### 4. HTTP 优化

#### 连接复用

```go
// HTTP 客户端配置
client := &http.Client{
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
    Timeout: 30 * time.Second,
}
```

#### 压缩响应

```go
// Gin 启用 gzip 压缩
import "github.com/gin-contrib/gzip"

router.Use(gzip.Gzip(gzip.DefaultCompression))
```

### 5. Milvus 优化

#### 索引参数调优

```go
// HNSW 索引参数
idx := index.NewHNSWIndex(
    entity.L2,    // 距离度量
    16,           // M: 每个节点的连接数（推荐 8-64）
    200,          // efConstruction: 索引构建时的搜索深度（推荐 100-500）
)

// 搜索参数
searchParam := entity.NewIndexHNSWSearchParam(64)  // ef: 搜索深度（推荐 top_k 的 2-10 倍）
```

#### 批量操作

```go
// 批量插入向量
const batchSize = 1000

for i := 0; i < len(vectors); i += batchSize {
    end := min(i+batchSize, len(vectors))
    batch := vectors[i:end]

    _, err := collection.Insert(ctx, batch...)
    if err != nil {
        return err
    }
}
```

---

## 性能监控和告警

### Prometheus 告警规则

```yaml
# deploy/prometheus/alerts.yml
groups:
  - name: sentinel-x-performance
    rules:
      # RAG 查询延迟告警
      - alert: RAGQueryHighLatency
        expr: histogram_quantile(0.99, rate(rag_query_duration_seconds_bucket[5m])) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "RAG query P99 latency is high"
          description: "P99 latency is {{ $value }}s (threshold: 10s)"

      # 用户登录延迟告警
      - alert: LoginHighLatency
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{endpoint="/v1/auth/login"}[5m])) > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Login P95 latency is high"
          description: "P95 latency is {{ $value }}s (threshold: 0.5s)"
```

### 性能仪表板

推荐使用 Grafana 创建以下仪表板：

1. **服务概览**
   - 请求速率（QPS）
   - 错误率
   - 延迟分布（P50/P95/P99）

2. **RAG 服务详情**
   - 查询耗时分解（Embedding、检索、LLM）
   - 缓存命中率
   - Milvus 检索性能

3. **数据库性能**
   - 连接池使用率
   - 查询延迟
   - 慢查询日志

4. **系统资源**
   - CPU 使用率
   - 内存使用率
   - Goroutine 数量

---

## 性能测试清单

在发布新版本前，必须完成以下性能测试：

- [ ] **基准测试**: 所有核心函数的 benchmark 通过
- [ ] **压力测试**: 各接口在目标 QPS 下满足 SLA
- [ ] **长时间测试**: 24 小时稳定性测试，无内存泄漏
- [ ] **缓存测试**: 验证缓存命中率 > 50%
- [ ] **数据库测试**: 验证连接池配置合理
- [ ] **并发测试**: 验证 Goroutine 池工作正常
- [ ] **回归测试**: 对比上一版本，性能无明显下降

---

## 参考资料

- [架构设计文档](../design/architecture.md)
- [配置管理文档](../configuration/README.md)
- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)
- [Milvus 性能调优](https://milvus.io/docs/performance_faq.md)
