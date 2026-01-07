# 测试文件迁移上下文摘要

生成时间: 2026-01-07 14:30:00

## 1. 任务目标

将所有测试文件从使用 `transport.Context` 迁移到 `*gin.Context`,修复编译错误。

## 2. 相似实现分析

### 现有Gin测试模式
参考文件: `internal/user-center/router/health_test.go`
```go
func TestHealthCheck(t *testing.T) {
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)

    // 创建请求
    req, _ := http.NewRequest("GET", "/health", nil)
    c.Request = req

    // 调用handler
    handler.HealthCheck(c)

    // 验证响应
    assert.Equal(t, 200, w.Code)
}
```

### 中间件测试模式
```go
func TestMiddleware(t *testing.T) {
    w := httptest.NewRecorder()
    c, r := gin.CreateTestContext(w)

    // 注册中间件
    r.Use(middleware)
    r.GET("/test", func(c *gin.Context) {
        c.Status(200)
    })

    // 执行请求
    req, _ := http.NewRequest("GET", "/test", nil)
    r.ServeHTTP(w, req)

    // 验证结果
    assert.Equal(t, 200, w.Code)
}
```

## 3. 需要修复的文件分类

### A. Handler测试 (3个文件)
- `internal/user-center/handler/api_test.go`
- `internal/user-center/handler/validation_test.go`
- `internal/user-center/handler/grpc_test.go`

### B. 中间件测试 (20+个文件)
- `pkg/infra/middleware/benchmark_test.go`
- `pkg/infra/middleware/auth/auth_test.go`
- `pkg/infra/middleware/security/cors_test.go`
- `pkg/infra/middleware/observability/integration_test.go`
- `pkg/infra/middleware/observability/tracing_test.go`
- `pkg/infra/middleware/resilience/body_limit_test.go`
- 等等...

### C. 工具测试 (4个文件)
- `pkg/utils/errors/example_test.go`
- `pkg/utils/errors/builder_test.go`
- `pkg/utils/errors/errno_test.go`
- `pkg/utils/validator/errors_test.go`

## 4. 迁移模式

### 模式1: 替换导入
```go
// 删除
import custom_http "github.com/kart-io/sentinel-x/pkg/infra/server/transport/http"
import "github.com/kart-io/sentinel-x/pkg/infra/server/transport"

// 添加
import "github.com/gin-gonic/gin"
import "net/http/httptest"
```

### 模式2: 创建测试上下文
```go
// 旧方式
ctx := custom_http.NewRequestContext(req, rec)

// 新方式
w := httptest.NewRecorder()
c, _ := gin.CreateTestContext(w)
c.Request = req
```

### 模式3: 中间件签名
```go
// 旧签名
func(next transport.HandlerFunc) transport.HandlerFunc

// 新签名
func(c *gin.Context)
```

### 模式4: 响应验证
```go
// 使用httptest.ResponseRecorder验证
assert.Equal(t, 200, w.Code)
assert.Contains(t, w.Body.String(), "expected")
```

## 5. 验证策略

- 每个文件修复后运行单独的测试: `go test -v ./path/to/package`
- 所有文件修复后运行完整测试: `make test`
- 确保测试覆盖率不降低

## 6. 风险点

- 可能存在隐式依赖 `transport.Context` 的方法调用
- 某些测试可能依赖旧的响应结构
- 中间件链式调用需要特别注意执行顺序
