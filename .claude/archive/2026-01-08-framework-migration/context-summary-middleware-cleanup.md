# 项目上下文摘要（中间件兼容性代码清理）
生成时间: 2026-01-06 14:30:00

## 1. 相似实现分析

### 已完成清理的中间件示例
- **Timeout**: pkg/infra/middleware/resilience/timeout.go
  - 模式：删除 Config 结构体和 WithConfig 函数
  - 统一接口：WithOptions(opts mwopts.TimeoutOptions)
  - 需注意：Options 定义在 pkg/options/middleware/

- **Recovery**: pkg/infra/middleware/core/recovery.go
  - 模式：删除 Config 结构体和 WithConfig 函数
  - 统一接口：WithOptions(opts mwopts.RecoveryOptions)
  - 需注意：保持简单函数（如 Recovery()）用于默认配置

## 2. 项目约定

### 命名约定
- 中间件函数：`MiddlewareNameWithOptions(opts mwopts.XxxOptions) HandlerFunc`
- 路由注册：`RegisterXxxRoutesWithOptions(router Router, opts mwopts.XxxOptions) error`
- Options 结构：`pkg/options/middleware/xxx.go`

### 文件组织
- 中间件实现：`pkg/infra/middleware/{category}/{name}.go`
- Options 定义：`pkg/options/middleware/{name}.go`
- 测试文件：`pkg/infra/middleware/{category}/{name}_test.go`

### 代码风格
- 删除所有 Config 结构体
- 删除所有 WithConfig 函数
- 删除所有 DefaultConfig 函数
- 保留简单工厂函数用于默认配置（可选）

## 3. 可复用组件清单

- `pkg/options/middleware`: Options 结构体定义
- `pkg/infra/middleware/types.go`: HandlerFunc 和 Router 接口
- `pkg/infra/pool`: 并发控制（必须使用）

## 4. 测试策略

### 测试框架
- Go testing + Testify
- 测试模式：单元测试
- 参考文件：pkg/infra/middleware/core/recovery_test.go

### 覆盖要求
- 测试 WithOptions 函数
- 测试默认配置函数
- 删除所有 WithConfig 测试

## 5. 依赖和集成点

### 外部依赖
- Gin, Echo (适配器)
- Prometheus (metrics)
- Redis (rate limit, 可选)

### 内部依赖
- pkg/options/middleware: Options 定义
- pkg/infra/server: Server 集成
- pkg/infra/pool: 并发控制

### 配置来源
- configs/*.yaml: 中间件配置读取后转换为 Options

## 6. 技术选型理由

### 为什么统一使用 WithOptions
- 与 server.ApplyMiddleware 集成一致
- 避免两套配置体系（Config vs Options）
- 简化 API 接口

### 优势
- 统一配置管理
- 减少重复代码
- 更好的类型安全

### 劣势和风险
- 需要迁移现有代码
- 可能影响使用者

## 7. 关键风险点

### 并发问题
- 健康检查并发访问 stores（必须使用 pool.SubmitToType）

### 边界条件
- 空指针处理（Options 字段可能为 nil）
- 零值处理（端口 0、超时 0）

### 性能瓶颈
- Metrics 收集（高频调用）
- Rate Limit 检查（高并发）

### 安全考虑
- CORS 配置错误可能导致安全问题
- Security Headers 缺失可能导致 XSS/Clickjacking
