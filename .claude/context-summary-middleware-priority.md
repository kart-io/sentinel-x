# 项目上下文摘要（中间件优先级机制）

生成时间：2026-01-06 22:30:00

## 1. 相似实现分析

### 实现1：pkg/options/middleware/registry.go
- 模式：注册器模式，用于管理中间件配置工厂函数
- 可复用：Registry 结构、Register/Create 模式、线程安全的 sync.RWMutex
- 需注意：已存在命名冲突检测、支持 CreateAll 批量创建

### 实现2：pkg/infra/middleware/middleware.go
- 模式：Chain 函数，按顺序组合多个中间件
- 可复用：中间件组合模式（从后向前应用）
- 需注意：中间件执行顺序由提供顺序决定

### 实现3：pkg/infra/server/transport/http/server.go:applyMiddleware
- 模式：手动按顺序调用 router.Use() 应用中间件
- 可复用：中间件启用检查模式（opts.IsEnabled）
- 需注意：
  - Recovery 必须最先（行211-214）
  - RequestID 必须第二（行217-220）
  - Logger 依赖 RequestID（行223-226）
  - Auth 必须在业务逻辑前（行244-254）
  - 顺序完全硬编码，易出错

## 2. 项目约定

### 命名约定
- 包名使用小写单词（middleware, resilience, observability）
- 类型名使用大驼峰（MiddlewareRegistrar, PriorityLevel）
- 常量使用大驼峰或全大写（PriorityRecovery, HeaderXRequestID）
- 接口以 -er 结尾（Router, Validator）

### 文件组织
- pkg/infra/middleware：中间件核心功能
- pkg/infra/middleware/{子包}：分类中间件（auth, resilience, security, observability）
- pkg/options/middleware：中间件配置选项
- pkg/infra/server/transport：传输层抽象

### 代码风格
- 使用 gofmt/gofumpt 格式化
- 导入顺序：标准库 -> 第三方库 -> 项目库
- 注释必须使用中文
- 错误处理优先检查

## 3. 可复用组件清单

- `pkg/infra/server/transport.Router`：路由器接口（Use方法）
- `pkg/infra/server/transport.MiddlewareFunc`：中间件函数签名
- `pkg/options/middleware.Options`：中间件配置管理
- `pkg/options/middleware.Registry`：注册器模式参考
- `sync.RWMutex`：线程安全保护
- `sort.Slice`：切片排序

## 4. 测试策略

### 测试框架
- Go testing + table-driven tests
- 不使用 testify（项目未引入）

### 测试模式
- 单元测试：测试排序逻辑、优先级分配
- 集成测试：测试中间件实际执行顺序
- Mock 实现：参考 pkg/infra/middleware/auth/auth_test.go 的 MockContext

### 参考文件
- pkg/infra/middleware/auth/auth_test.go：MockContext 实现
- pkg/infra/middleware/security/mock_test.go：Mock 模式

### 覆盖要求
- 排序逻辑覆盖率 > 90%
- 边界条件测试（空列表、同优先级、重复注册）

## 5. 依赖和集成点

### 外部依赖
- 标准库：sort, sync, fmt, errors

### 内部依赖
- pkg/infra/server/transport：Router 和 MiddlewareFunc 接口
- pkg/options/middleware：Options 配置管理
- pkg/infra/middleware：现有中间件实现

### 集成方式
- 在 server.go:applyMiddleware 中使用注册器
- 通过依赖注入传递 Options
- 保持向后兼容（可选功能）

### 配置来源
- pkg/options/middleware/options.go：中间件配置

## 6. 技术选型理由

### 为什么用优先级系统
- 当前手动顺序易出错（如把 Recovery 放到后面）
- 新增中间件时不知道插入位置
- 中间件依赖关系不明确（Logger 依赖 RequestID）
- 维护困难，需要记忆正确顺序

### 优势
- 自动管理顺序，减少人为错误
- 明确的依赖关系（通过优先级体现）
- 易于扩展（新增中间件只需指定优先级）
- 支持同优先级（按注册顺序）

### 劣势和风险
- 增加一层抽象复杂度
- 优先级常量需要精心设计
- 可能与现有代码集成需要调整

## 7. 关键风险点

### 并发问题
- 注册器需要线程安全（参考 Registry 使用 sync.RWMutex）
- 排序后的中间件列表应该只读

### 边界条件
- 空中间件列表
- 同优先级中间件（按注册顺序）
- 重复注册检测
- 优先级值冲突

### 性能瓶颈
- 排序操作（在 Apply 时一次性完成，启动时开销可接受）
- 避免在每次请求时排序

### 安全考虑
- Recovery 必须最先执行（最高优先级）
- Auth 必须在业务逻辑前（较低优先级）
- 不允许运行时修改优先级（防止绕过安全中间件）

## 8. 实现计划

### 新增文件
- `pkg/infra/middleware/priority.go`：优先级常量和注册器

### 修改文件
- `pkg/infra/server/transport/http/server.go`：使用注册器替代手动应用

### 测试文件
- `pkg/infra/middleware/priority_test.go`：单元测试

### 优先级定义（数值越大优先级越高）
```
PriorityRecovery    = 1000  // 最高优先级，必须第一个
PriorityRequestID   = 900   // 第二，为其他中间件提供 RequestID
PriorityLogger      = 800   // 依赖 RequestID
PriorityMetrics     = 700   // 观测性
PriorityTracing     = 650   // 观测性
PriorityCORS        = 600   // 安全相关
PrioritySecurityHeaders = 550
PriorityTimeout     = 500   // 弹性机制
PriorityAuth        = 400   // 认证，必须在业务逻辑前
PriorityAuthz       = 300   // 授权，在认证后
PriorityCustom      = 100   // 自定义中间件默认优先级
```

### 向后兼容性
- 保留原有的 applyMiddleware 逻辑作为备选
- 优先级机制作为可选功能
- 提供迁移指南
