# 项目上下文摘要（BodyLimit 和 Compression 中间件）

生成时间：2026-01-06 17:30:00

## 1. 相似实现分析

### 实现1: pkg/options/middleware/timeout.go
- 模式：Options 结构体模式
- 可复用：
  - `MiddlewareConfig` 接口（Validate/Complete/AddFlags）
  - `PathMatcher` 结构体（SkipPaths 模式）
  - `Register()` 函数注册到全局注册表
- 需注意：
  - 必须实现 `MiddlewareConfig` 接口的所有方法
  - 在 `init()` 函数中注册到全局注册表
  - 使用 `mapstructure` 标签支持配置文件映射

### 实现2: pkg/infra/middleware/resilience/timeout.go
- 模式：中间件函数模式
- 可复用：
  - `TimeoutWithOptions(opts mwopts.TimeoutOptions)` 构造函数模式
  - `pool.SubmitToType()` 并发池提交模式
  - 降级处理：池不可用时同步执行
- 需注意：
  - 使用 ants 池而非直接创建 goroutine
  - 提供降级方案（池失败时的处理）
  - Panic 恢复机制

### 实现3: pkg/options/middleware/cors.go
- 模式：配置选项模式
- 可复用：
  - `NewXXXOptions()` 工厂函数模式
  - `AddFlags()` 命令行参数绑定
  - `Validate()` 验证逻辑
- 需注意：
  - 提供合理的默认值
  - 使用 `options.Join(prefixes...)` 拼接参数前缀
  - 返回错误切片而非单个错误

## 2. 项目约定

### 命名约定
- Options 结构体：`XXXOptions`（如 `BodyLimitOptions`）
- 工厂函数：`NewXXXOptions()`
- 中间件函数：`XXXWithOptions(opts mwopts.XXXOptions)`
- 目录命名：按功能分类（`resilience`、`performance`、`observability` 等）

### 文件组织
- `pkg/options/middleware/` - 配置选项定义
- `pkg/infra/middleware/{category}/` - 中间件实现
  - `resilience/` - 弹性机制（recovery, timeout, ratelimit）
  - `performance/` - 性能优化（需新建目录）
  - `security/` - 安全相关（cors 等）
  - `observability/` - 可观测性（logger, metrics, tracing）

### 导入顺序
1. 标准库（如 `net/http`, `compress/gzip`）
2. 第三方库（如 `github.com/spf13/pflag`）
3. 项目内部库
   - `github.com/kart-io/logger`
   - `github.com/kart-io/sentinel-x/pkg/...`

### 代码风格
- 使用中文注释
- 每个导出的类型/函数必须有文档注释
- 测试文件与实现文件同目录
- 测试函数命名：`Test{FunctionName}_{Scenario}`

## 3. 可复用组件清单

### pkg/options/middleware/
- `MiddlewareConfig` 接口 - 统一配置接口
- `PathMatcher` 结构体 - 路径匹配（SkipPaths/SkipPathPrefixes）
- `Register()` 函数 - 注册到全局注册表
- `options.Join()` 函数 - 拼接命令行参数前缀

### pkg/infra/middleware/
- `PriorityXXX` 常量 - 优先级定义
- `transport.MiddlewareFunc` 类型 - 中间件函数签名
- `transport.Context` 接口 - 统一上下文

### pkg/infra/pool/
- `pool.SubmitToType()` - 提交任务到指定类型的池
- `pool.TimeoutPool` - 超时专用池（容量 5000）
- 降级模式：池不可用时的处理策略

## 4. 测试策略

### 测试框架
- Go 标准 `testing` 包
- 使用 `httptest` 进行 HTTP 测试
- 使用 `sync.WaitGroup` 处理并发测试

### 测试模式
- 表驱动测试（table-driven tests）
- Mock Context 测试（如 `newMockContext()`）
- 边界条件测试（零值、超大值、负值）

### 参考文件
- `pkg/infra/middleware/resilience/timeout_test.go`
- 测试场景：
  - 正常请求
  - 慢请求（超时）
  - 跳过路径
  - 并发请求
  - Panic 恢复
  - Goroutine 泄漏

### 覆盖要求
- 核心逻辑覆盖率 > 80%
- 必须测试边界条件
- 必须测试错误场景

## 5. 依赖和集成点

### 外部依赖
- `compress/gzip` - 标准库压缩
- `net/http` - HTTP 标准库
- `github.com/spf13/pflag` - 命令行参数

### 内部依赖
- `github.com/kart-io/logger` - 日志记录
- `github.com/kart-io/sentinel-x/pkg/infra/pool` - 并发池
- `github.com/kart-io/sentinel-x/pkg/infra/server/transport` - 统一传输层
- `github.com/kart-io/sentinel-x/pkg/options` - 选项工具
- `github.com/kart-io/sentinel-x/pkg/utils/errors` - 错误定义
- `github.com/kart-io/sentinel-x/pkg/utils/response` - 响应工具

### 集成方式
- 通过 `pkg/options/middleware/options.go` 统一管理
- 通过 `pkg/infra/middleware/priority.go` 定义优先级
- 通过 `internal/*/server.go` 集成到服务

### 配置来源
- YAML 配置文件（通过 Viper）
- 命令行参数（通过 pflag）
- 环境变量（通过 Viper）

## 6. 技术选型理由

### BodyLimit
- **为什么需要**：防止 DoS 攻击，限制请求体大小
- **实现方式**：使用 `http.MaxBytesReader` 限制读取
- **优势**：
  - 标准库实现，无额外依赖
  - 早期拦截，节省资源
  - 支持跳过特定路径
- **风险**：
  - 必须在 CORS 之后（优先级 550）
  - 必须在 Timeout 之前（优先级 500）

### Compression
- **为什么需要**：减少带宽消耗，提升性能
- **实现方式**：使用 `compress/gzip` 包装 ResponseWriter
- **优势**：
  - 标准库实现，无额外依赖
  - 可配置压缩级别和类型
  - 支持跳过路径
- **风险**：
  - 必须在业务逻辑之后（优先级 200）
  - 可能增加 CPU 消耗
  - 小响应体不应压缩

## 7. 关键风险点

### 并发问题
- BodyLimit：无并发风险（同步读取）
- Compression：ResponseWriter 替换必须线程安全

### 边界条件
- BodyLimit：Content-Length 缺失时的处理
- Compression：Accept-Encoding 缺失或不支持 gzip
- MinSize：小于阈值的响应体不压缩

### 性能瓶颈
- Compression：高压缩级别（9）可能消耗大量 CPU
- 建议默认中等级别（6）平衡性能和压缩率

### 安全考虑
- BodyLimit：必须处理恶意超大 Content-Length 头
- Compression：避免压缩已压缩内容（如图片）

## 8. 实现计划

### 新增文件
1. `pkg/options/middleware/body_limit.go`
2. `pkg/options/middleware/compression.go`
3. `pkg/infra/middleware/resilience/body_limit.go`
4. `pkg/infra/middleware/resilience/body_limit_test.go`
5. `pkg/infra/middleware/performance/compression.go`（新建目录）
6. `pkg/infra/middleware/performance/compression_test.go`

### 修改文件
1. `pkg/options/middleware/options.go`
   - 添加 `BodyLimit` 和 `Compression` 字段
   - 更新 `NewOptions()`, `Validate()`, `Complete()`, `AddFlags()`
   - 添加常量和辅助函数
2. `pkg/infra/middleware/priority.go`
   - 添加 `PriorityBodyLimit = 550`
   - 添加 `PriorityCompression = 200`

### 优先级设计
```
1000 Recovery
 900 RequestID
 800 Logger
 700 Metrics
 650 Tracing
 600 CORS
 550 BodyLimit   ← 新增（在 CORS 之后，SecurityHeaders 同级）
 500 Timeout
 400 Auth
 300 Authz
 200 Compression ← 新增（业务逻辑之后）
 100 Custom
```

### 验证清单
- [ ] BodyLimit 超大请求被拒绝
- [ ] BodyLimit 跳过路径正常工作
- [ ] Compression 响应被正确压缩
- [ ] Compression 小响应体不压缩
- [ ] Compression 不支持 gzip 的客户端不压缩
- [ ] 两个中间件与其他中间件兼容
- [ ] 配置可通过 YAML/命令行/环境变量加载
