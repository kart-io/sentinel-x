## 项目上下文摘要（中间件测试文件迁移）
生成时间：2026-01-07

### 1. 相似实现分析
- **实现1**: pkg/infra/middleware/auth/auth_test.go:46-51
  - 模式：使用 `gin.CreateTestContext()` 创建测试上下文
  - 可复用：`w := httptest.NewRecorder(); c, _ := gin.CreateTestContext(w); c.Request = req`
  - 需注意：Gin context 使用 `c.Request` 而非 `HTTPRequest()`

- **实现2**: pkg/infra/middleware/security/cors_test.go:25-36
  - 模式：集成测试使用完整 gin router
  - 可复用：`w := httptest.NewRecorder(); c, r := gin.CreateTestContext(w); r.Use(middleware); r.ServeHTTP(w, req)`
  - 需注意：Preflight 请求处理和 OPTIONS 方法

### 2. 项目约定
- **命名约定**: 测试函数使用 `Test` 前缀，Benchmark 使用 `Benchmark` 前缀
- **文件组织**: 测试文件与源文件同目录，使用 `_test.go` 后缀
- **导入顺序**: 标准库 -> 第三方库（gin） -> 项目库
- **代码风格**: 简体中文注释，gofmt 格式化

### 3. 需要迁移的文件清单
- `pkg/infra/middleware/benchmark_test.go`: 所有 benchmark 函数（使用 mockContext）
- `pkg/infra/middleware/observability/integration_test.go`: 集成测试（使用 mockContext）
- `pkg/infra/middleware/observability/tracing_test.go`: tracing 测试（使用 mockTracingContext 和 transport.Context）
- `pkg/infra/middleware/resilience/body_limit_test.go`: body limit 测试（使用 mockContext）

### 4. 迁移模式

#### 模式1: 基础测试（handler 签名迁移）
```go
// 旧
func handler(c transport.Context) { ... }

// 新
func handler(c *gin.Context) { ... }
```

#### 模式2: 测试上下文创建
```go
// 旧
mockCtx := newMockContext(req, w)

// 新
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request = req
```

#### 模式3: 集成测试
```go
// 新
w := httptest.NewRecorder()
c, r := gin.CreateTestContext(w)
r.Use(middleware)
r.GET("/test", handler)
r.ServeHTTP(w, req)
```

#### 模式4: 导入更新
```go
// 移除
import "github.com/kart-io/sentinel-x/pkg/infra/server/transport"

// 添加
import "github.com/gin-gonic/gin"
```

### 5. 依赖和集成点
- **外部依赖**: Gin, net/http/httptest
- **内部依赖**: pkg/infra/middleware各子包, pkg/options/middleware
- **集成方式**: 中间件适配 gin.HandlerFunc
- **配置来源**: pkg/options/middleware

### 6. 技术选型理由
- **为什么用 Gin**: 项目已迁移到 Gin 作为统一 HTTP 框架
- **优势**: Gin 性能高，社区活跃，测试工具完善
- **劣势和风险**: 需要彻底移除旧 transport.Context 抽象层

### 7. 关键风险点
- **并发问题**: Benchmark 测试中的并发访问
- **边界条件**: 空请求、空 body、超大 body
- **性能瓶颈**: Benchmark 测试需要保持测量准确性
- **安全考虑**: Body limit 保护、请求大小限制

### 8. 特定文件注意事项

#### benchmark_test.go
- 所有 benchmark 函数需要更新中间件调用方式
- 使用 `gin.CreateTestContext()` 替代 `newMockContext()`
- 保持 benchmark 测量逻辑不变（ResetTimer, ReportAllocs）
- 中间件链构造改为 Gin 方式

#### observability/tracing_test.go
- SpanNameFormatter: `func(_ transport.Context) string` → `func(ctx *gin.Context) string`
- AttributeExtractor: `func(_ transport.Context) []attribute.KeyValue` → `func(ctx *gin.Context) []attribute.KeyValue`
- mockTracingContext 可以删除，直接使用 gin.Context

#### observability/integration_test.go
- 使用完整 gin router 进行集成测试
- 移除 mockContext，使用 `gin.CreateTestContext()`

#### resilience/body_limit_test.go
- 更新所有测试中的 handler 函数签名
- 移除 mockContext 依赖
- 保持 body size 测试逻辑
