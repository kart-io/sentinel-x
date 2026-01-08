## 项目上下文摘要（Request ID 测试迁移）
生成时间：2026-01-07 17:30:00

### 1. 相似实现分析
- **实现1**: pkg/infra/middleware/security/cors_test.go:14-63
  - 模式：使用 gin.CreateTestContext 创建测试上下文
  - 可复用：`w := httptest.NewRecorder(); _, r := gin.CreateTestContext(w)` 模式
  - 需注意：使用 `w.Header().Get()` 获取响应头，而不是 mockCtx.headers

- **实现2**: pkg/infra/middleware/security/cors_test.go:65-99
  - 模式：捕获 handler 调用状态
  - 可复用：`handlerCalled := false` 模式
  - 需注意：在 handler 中设置标志位验证是否被调用

- **实现3**: pkg/infra/middleware/request_id.go:58-74
  - 模式：Gin 中间件实现
  - 可复用：理解中间件如何设置 context 和 header
  - 需注意：使用 `c.GetHeader()`、`c.Header()`、`c.Request.WithContext()`

### 2. 项目约定
- **命名约定**: 测试函数使用 Test 前缀，表格驱动测试使用 tests 切片
- **文件组织**: 测试文件与源文件放在同一包中
- **导入顺序**: 标准库 -> 第三方库 -> 项目库
- **代码风格**: 使用简体中文注释

### 3. 可复用组件清单
- `github.com/gin-gonic/gin`: Gin 框架（替换 transport.Context）
- `net/http/httptest`: HTTP 测试工具
- `requestutil.GetRequestID`: 从 context 获取 request ID
- `requestutil.GenerateRequestID`: 生成 request ID

### 4. 测试策略
- **测试框架**: Go testing
- **测试模式**: 单元测试 + 表格驱动测试
- **参考文件**: pkg/infra/middleware/security/cors_test.go
- **覆盖要求**: 所有功能分支覆盖

### 5. 依赖和集成点
- **外部依赖**: github.com/gin-gonic/gin, net/http, net/http/httptest
- **内部依赖**: pkg/infra/middleware/requestutil, pkg/options/middleware
- **集成方式**: 中间件函数调用
- **配置来源**: mwopts.RequestIDOptions

### 6. 技术选型理由
- **为什么从 mockContext 迁移到 gin.CreateTestContext**: 统一测试模式，使用真实的 Gin 上下文
- **优势**: 更接近真实运行环境，减少 mock 代码维护
- **劣势和风险**: 需要重写所有使用 mockContext 的测试

### 7. 关键风险点
- **迁移问题**: 确保所有测试逻辑保持一致，不遗漏边界条件
- **边界条件**: header 获取方式改变、context 捕获方式改变
- **测试覆盖**: 确保迁移后所有测试仍然有效
- **代码一致性**: 遵循已迁移文件的模式

### 8. 迁移模式总结

#### 旧模式（mockContext）
```go
handler := middleware(func(_ transport.Context) {})
mockCtx := newMockContext(req, w)
handler(mockCtx)
requestID := mockCtx.headers[HeaderXRequestID]
```

#### 新模式（gin.CreateTestContext）
```go
w := httptest.NewRecorder()
_, r := gin.CreateTestContext(w)
r.Use(middleware)
r.GET("/test", func(_ *gin.Context) {})
r.ServeHTTP(w, req)
requestID := w.Header().Get(HeaderXRequestID)
```

#### 捕获 context 的新模式
```go
var capturedCtx context.Context
w := httptest.NewRecorder()
_, r := gin.CreateTestContext(w)
r.Use(middleware)
r.GET("/test", func(c *gin.Context) {
    capturedCtx = c.Request.Context()
})
r.ServeHTTP(w, req)
requestID := requestutil.GetRequestID(capturedCtx)
```

### 9. 需要修复的文件清单
1. pkg/infra/middleware/request_id_test.go（11个测试函数）
2. pkg/infra/middleware/request_id_integration_test.go（5个测试函数）

### 10. 修复步骤
1. 更新导入：移除 transport，添加 gin
2. 替换测试上下文创建方式
3. 替换 header 获取方式
4. 替换 context 捕获方式
5. 运行测试验证
