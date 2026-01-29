## 项目上下文摘要（去除 Echo 框架）
生成时间：2026-01-15 23:00:00

### 1. 项目现状分析

**重要发现：项目实际只使用了 Gin 框架，并未实现 Echo 支持**

#### 1.1 实际使用情况

**Gin 框架的使用（已实际实现）：**
- **依赖声明**：`go.mod:10` - `github.com/gin-gonic/gin v1.11.0`
- **核心服务器实现**：`pkg/infra/server/transport/http/server.go`
  - 直接使用 `gin.Engine` 创建引擎
  - 不使用默认中间件，自定义中间件链
  - 通过工厂模式管理中间件注册和应用
- **中间件系统**：`pkg/infra/middleware/` 目录下所有中间件都是基于 Gin 实现
  - Recovery、RequestID、Logger、CORS、Timeout
  - Auth、Authz、RateLimit、CircuitBreaker
  - Tracing、Metrics、Compression、SecurityHeaders
- **Handler 层**：所有 handler 都使用 `*gin.Context`
  - `internal/user-center/handler/` - UserHandler、AuthHandler、RoleHandler
  - `internal/api/handler/` - DemoHandler
  - `internal/rag/handler/` - RAGHandler
- **路由注册**：`internal/user-center/router/router.go:28` - 直接使用 `gin.Engine`
- **响应处理**：`pkg/utils/response/writer.go` - 所有方法都是基于 `*gin.Context`

**Echo 框架的使用（仅存在于文档和 go.sum）：**
- **依赖记录**：`go.sum:1114-1116` - 有 Echo 的版本记录，但 go.mod 中未声明
- **文档提及**：
  - `docs/project-analysis.md:610` - 依赖列表中提到 Echo
  - `docs/design/user-center.md:117,336,750` - 架构图中显示 Echo
  - `docs/design/architecture.md:169` - 架构图中显示 echo.Echo
- **实际代码**：没有任何实际使用 Echo 的代码

#### 1.2 适配器模式分析

**文档中的设计（未实现）：**
- 设计文档提到使用适配器模式支持 Gin 和 Echo 切换
- 计划通过 `transport.Context` 接口实现框架无关
- 配置文件中可以通过 `server.http.adapter: gin/echo` 切换

**实际代码中的情况：**
- 没有适配器目录：`pkg/infra/adapter/` 不存在
- 没有适配器代码：搜索 `bridge|Bridge` 无匹配
- 没有适配器常量：搜索 `AdapterGin|AdapterEcho` 无匹配
- 没有框架选择机制：所有代码直接使用 Gin，没有切换逻辑

### 2. 项目约定

- **命名约定**：包名使用小写单数形式，接口名通常加 `er` 后缀
- **文件组织**：
  - `internal/` - 私有业务代码
  - `pkg/` - 公共库代码
  - `cmd/` - 应用程序入口
- **代码风格**：
  - 使用 `gofumpt` 格式化
  - 严格的错误处理规范
  - 统一的响应格式（`pkg/utils/response`）

### 3. 可复用组件清单

- `pkg/infra/server/transport/http/server.go` - HTTP 服务器核心
- `pkg/infra/middleware/` - 完整的中间件系统（基于 Gin）
- `pkg/options/middleware/` - 中间件配置和工厂模式
- `pkg/utils/response/writer.go` - 响应写入工具
- `pkg/utils/errors/` - 统一错误处理
- `pkg/security/auth/` - JWT 认证
- `pkg/security/authz/` - Casbin 授权

### 4. 测试策略

- **测试框架**：Go testing + Testify
- **测试模式**：单元测试 + 集成测试
- **Mock 策略**：使用 `gin.CreateTestContext` 创建测试上下文
- **覆盖要求**：核心业务逻辑覆盖率 > 80%

### 5. 依赖和集成点

- **外部依赖**：Gin v1.11.0, GORM, Redis, JWT, Casbin
- **内部依赖**：
  - `pkg/infra/server` - 服务器抽象
  - `pkg/infra/middleware` - 中间件系统
  - `pkg/security` - 认证授权
- **集成方式**：依赖注入 + 接口调用
- **配置来源**：`configs/` 目录下的 YAML 文件

### 6. 技术选型理由

- **为什么用 Gin**：
  - 高性能：比标准库路由快
  - 生态成熟：丰富的中间件和工具
  - 社区活跃：持续维护和更新
  - 学习成本低：API 简洁直观
- **优势**：
  - 快速开发：开箱即用的中间件
  - 高度可定制：灵活的中间件链
  - 类型安全：强类型的路由和参数绑定
- **劣势**：
  - 功能单一：仅支持 HTTP（无 WebSocket 等扩展）
  - 标准化不足：某些实现偏离标准库

### 7. 关键风险点

- **并发问题**：中间件链中的 Context 共享
- **边界条件**：参数验证失败、空指针、超时处理
- **性能瓶颈**：慢查询、锁竞争、内存泄漏
- **安全考虑**：SQL注入、XSS、越权访问

### 8. 重要结论

**项目实际上不存在"同时支持 Gin 和 Echo"的代码实现**：
1. Echo 仅存在于文档设计说明中，未实际编码
2. 所有代码都是基于 Gin 的直接实现
3. 没有适配器模式、没有框架切换机制
4. `go.sum` 中的 Echo 依赖可能是历史遗留或间接依赖

**因此，"去除 Echo 框架"的任务实际上只需要：**
1. 清理文档中关于 Echo 的提及
2. 清理 `go.sum` 中的 Echo 依赖记录（如果确认是间接依赖则保留）
3. 更新架构文档，移除适配器模式的描述
4. 确认 Gin 实现的代码质量符合标准
