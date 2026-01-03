# 中间件配置架构分析与改进方案

## 执行摘要

当前项目的中间件配置架构存在明显的**扩展性问题**，违反了**开闭原则**（对扩展开放，对修改关闭）。每次新增中间件需要修改多达 **8-10 个位置**的代码，增加维护成本和出错风险。

本报告分析了当前设计的优缺点，参考了业界最佳实践，提出了 **3 个改进方案**，推荐采用**方案 B：注册器模式**，可将新增中间件的修改点从 8-10 个减少到 **2-3 个**，同时保持类型安全和配置灵活性。

---

## 1. 当前架构分析

### 1.1 架构概览

当前中间件配置采用**显式字段结构**设计：

```
pkg/options/middleware/
├── options.go          # 主配置结构，包含所有中间件字段
├── merge.go            # 合并逻辑，包含 MergeSubOptions 方法
├── cors.go             # CORS 中间件配置
├── health.go           # Health 中间件配置
├── metrics.go          # Metrics 中间件配置
├── logger.go           # Logger 中间件配置
├── recovery.go         # Recovery 中间件配置
├── request_id.go       # RequestID 中间件配置
├── timeout.go          # Timeout 中间件配置
├── pprof.go            # Pprof 中间件配置
└── auth.go             # Auth/Authz 中间件配置

internal/api/options.go          # API 服务配置（调用 MergeSubOptions）
internal/user-center/options.go  # 用户中心服务配置（调用 MergeSubOptions）
internal/rag/options.go          # RAG 服务配置（调用 MergeSubOptions）
```

**核心结构**：

```go
// pkg/options/middleware/options.go
type Options struct {
    Enabled []string              // 启用的中间件列表

    // 每个中间件的配置字段（硬编码）
    Recovery  *RecoveryOptions
    RequestID *RequestIDOptions
    Logger    *LoggerOptions
    CORS      *CORSOptions
    Timeout   *TimeoutOptions
    Health    *HealthOptions
    Metrics   *MetricsOptions
    Pprof     *PprofOptions
    Auth      *AuthOptions
    Authz     *AuthzOptions

    enabledSet map[string]bool    // 内部缓存
}
```

### 1.2 新增中间件的修改点（问题根源）

假设要新增一个 **RateLimit** 中间件，需要修改以下位置：

| 序号 | 文件路径 | 修改内容 | 代码行数 |
|------|---------|---------|---------|
| 1 | `pkg/options/middleware/options.go` | 添加常量 `MiddlewareRateLimit` | 1 行 |
| 2 | `pkg/options/middleware/options.go` | `AllMiddlewares` 数组添加元素 | 1 行 |
| 3 | `pkg/options/middleware/options.go` | `Options` 结构添加字段 `RateLimit *RateLimitOptions` | 1 行 |
| 4 | `pkg/options/middleware/options.go` | `ensureDefaults()` 方法添加初始化逻辑 | 3 行 |
| 5 | `pkg/options/middleware/options.go` | `Validate()` 方法添加验证逻辑 | 5 行 |
| 6 | `pkg/options/middleware/options.go` | `Complete()` 方法添加完成逻辑 | 3 行 |
| 7 | `pkg/options/middleware/options.go` | `AddFlags()` 方法添加标志注册 | 1 行 |
| 8 | `pkg/options/middleware/merge.go` | `Merge()` 方法添加合并逻辑 | 3 行 |
| 9 | `pkg/options/middleware/merge.go` | `MergeSubOptions()` **方法签名**添加参数 | 1 行 |
| 10 | `pkg/options/middleware/merge.go` | `MergeSubOptions()` 方法体添加合并调用 | 3 行 |
| 11 | `internal/api/options.go` | `Options` 结构添加 `RateLimit` 字段 | 1 行 |
| 12 | `internal/api/options.go` | `NewOptions()` 添加初始化 | 1 行 |
| 13 | `internal/api/options.go` | `Complete()` 中调用 `MergeSubOptions` 时添加参数 | 1 行 |
| 14 | `internal/user-center/options.go` | 重复步骤 11-13 | 3 行 |
| 15 | `internal/rag/options.go` | 重复步骤 11-13 | 3 行 |
| **总计** | - | **8-10 个文件，30+ 行代码** | - |

**关键问题**：
- **步骤 9** 是致命的：修改 `MergeSubOptions` 方法签名是**破坏性变更**，强制所有调用方同步修改
- **步骤 11-15** 造成**配置重复**：每个服务的 `options.go` 都有相同的字段定义
- 违反 **DRY 原则**（Don't Repeat Yourself）
- 违反 **OCP 原则**（Open-Closed Principle）

### 1.3 当前设计的优缺点

#### ✅ 优点

1. **类型安全**
   每个中间件配置都是强类型的结构体，编译时检查，避免运行时错误。

2. **IDE 友好**
   自动补全、重构、跳转等功能完美支持。

3. **配置清晰**
   一眼看出系统有哪些中间件，字段结构明确。

4. **验证完整**
   每个子选项都有独立的 `Validate()` 方法，便于细粒度验证。

5. **合并灵活**
   支持顶层配置与服务级配置的优先级合并（`Merge()` 和 `MergeSubOptions()`）。

#### ❌ 缺点

1. **扩展性极差**（致命缺陷）
   新增中间件需要修改 8-10 个文件，容易遗漏或出错。

2. **违反开闭原则**
   无法在不修改现有代码的前提下扩展中间件。

3. **方法签名污染**
   `MergeSubOptions()` 参数列表随中间件数量线性增长，当前已有 8 个参数，极限可能达到 20+ 个。

4. **配置重复**
   每个服务的 `options.go` 都有相同的中间件字段定义，违反 DRY 原则。

5. **维护成本高**
   中间件数量从当前的 10 个增长到 20 个时，维护成本将翻倍。

6. **测试复杂**
   每次新增中间件都需要更新多个测试文件。

---

## 2. 业界最佳实践参考

### 2.1 Gin 框架的中间件模式

Gin 采用**函数式中间件链**模式：

```go
// 使用方式
r := gin.Default()
r.Use(gin.Logger())
r.Use(gin.Recovery())
r.Use(cors.Default())
r.Use(middleware.RateLimit(100))  // 新增中间件，零修改
```

**优点**：极致的扩展性，无需修改框架代码
**缺点**：缺少集中配置，难以与 Viper/YAML 配置集成

### 2.2 Echo 框架的中间件模式

Echo 结合了链式调用和配置结构：

```go
e := echo.New()
e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
    Skipper: middleware.DefaultSkipper,
}))
```

**优点**：配置结构化，支持 YAML 集成
**缺点**：仍需硬编码中间件类型

### 2.3 Kratos 微服务框架

Kratos 使用**选项函数模式**：

```go
httpSrv := http.NewServer(
    http.Address(":8000"),
    http.Middleware(
        recovery.Recovery(),
        logging.Server(),
        metrics.Server(),
    ),
)
```

**优点**：简洁优雅，支持编程式配置
**缺点**：不适合复杂的 YAML 配置场景

### 2.4 核心设计模式总结

参考资料显示，Go 社区推荐的中间件配置模式包括：

1. **Functional Options Pattern**（功能选项模式）
   用函数修改配置对象，易于扩展，不破坏 API。

2. **Registry Pattern**（注册器模式）
   中间件通过名称注册到全局注册器，配置通过名称查找。

3. **Chain Pattern**（链式模式）
   中间件组合成链，顺序执行。

---

## 3. 改进方案对比

### 方案 A：动态配置（map[string]interface{}）

#### 设计思路

使用 `map` 存储中间件配置，通过反射或类型断言获取配置。

```go
type Options struct {
    Enabled []string
    Configs map[string]interface{}  // 动态配置
}

// 使用
opts.Configs["ratelimit"] = &RateLimitOptions{QPS: 100}
cfg := opts.Configs["ratelimit"].(*RateLimitOptions)  // 类型断言
```

#### 优缺点

| 维度 | 评分 | 说明 |
|------|------|------|
| **扩展性** | ⭐⭐⭐⭐⭐ | 新增中间件零修改 |
| **类型安全** | ⭐⭐ | 丢失编译时检查，依赖类型断言 |
| **IDE 支持** | ⭐ | 自动补全失效 |
| **YAML 支持** | ⭐⭐ | 需要自定义解析逻辑 |
| **可维护性** | ⭐⭐ | 运行时错误风险高 |
| **实现难度** | ⭐⭐⭐ | 需要大量反射代码 |

#### 工作量评估

- **修改文件数**：5-7 个
- **开发时间**：2-3 天
- **风险等级**：高（类型安全丧失）

---

### 方案 B：注册器模式 + 接口抽象（推荐）

#### 设计思路

定义统一的中间件配置接口，通过注册器管理，保留类型安全。

```go
// 1. 定义统一接口
type MiddlewareConfig interface {
    Validate() error
    Complete() error
    AddFlags(fs *pflag.FlagSet)
    MergeFrom(source MiddlewareConfig)
}

// 2. 注册器
type Registry struct {
    configs map[string]MiddlewareConfig
    factories map[string]func() MiddlewareConfig
}

var globalRegistry = NewRegistry()

func Register(name string, factory func() MiddlewareConfig) {
    globalRegistry.Register(name, factory)
}

// 3. 初始化时注册（在各中间件文件的 init() 函数中）
func init() {
    Register("ratelimit", func() MiddlewareConfig {
        return NewRateLimitOptions()
    })
}

// 4. 主配置结构
type Options struct {
    Enabled []string
    configs map[string]MiddlewareConfig  // 动态存储
}

// 5. 通用合并方法
func (o *Options) MergeAll(sources map[string]MiddlewareConfig) {
    for name, source := range sources {
        if cfg, ok := o.configs[name]; ok {
            cfg.MergeFrom(source)
        }
    }
}

// 6. 类型安全的访问器
func (o *Options) GetRateLimit() *RateLimitOptions {
    cfg, _ := o.configs["ratelimit"]
    return cfg.(*RateLimitOptions)  // 单一位置的类型断言
}
```

#### 新增中间件的步骤（仅 2-3 个修改点）

1. **创建 `ratelimit.go` 文件**
   ```go
   package middleware

   func init() {
       Register("ratelimit", func() MiddlewareConfig {
           return NewRateLimitOptions()
       })
   }

   type RateLimitOptions struct { ... }
   func (o *RateLimitOptions) Validate() error { ... }
   func (o *RateLimitOptions) Complete() error { ... }
   func (o *RateLimitOptions) MergeFrom(source MiddlewareConfig) { ... }
   ```

2. **（可选）在 `options.go` 添加类型安全访问器**
   ```go
   func (o *Options) GetRateLimit() *RateLimitOptions {
       return o.GetConfig("ratelimit").(*RateLimitOptions)
   }
   ```

3. **更新常量列表**（可通过注册器自动生成）
   ```go
   const MiddlewareRateLimit = "ratelimit"
   ```

**关键改进**：
- ❌ 不再需要修改 `MergeSubOptions()` 方法签名
- ❌ 不再需要修改各服务的 `options.go`
- ✅ 中间件自包含，符合**单一职责原则**
- ✅ 保留类型安全（通过访问器方法）

#### 优缺点

| 维度 | 评分 | 说明 |
|------|------|------|
| **扩展性** | ⭐⭐⭐⭐⭐ | 修改点从 8-10 个减少到 2-3 个 |
| **类型安全** | ⭐⭐⭐⭐ | 通过访问器方法保证类型安全 |
| **IDE 支持** | ⭐⭐⭐⭐ | 访问器方法支持自动补全 |
| **YAML 支持** | ⭐⭐⭐⭐ | 通过 `mapstructure` 自动解析 |
| **可维护性** | ⭐⭐⭐⭐⭐ | 中间件自包含，易于维护 |
| **实现难度** | ⭐⭐⭐⭐ | 需要重构现有代码，但逻辑清晰 |

#### 工作量评估

- **修改文件数**：8-12 个（一次性重构）
- **开发时间**：3-5 天
- **风险等级**：中（需要充分测试）

---

### 方案 C：代码生成 + 显式字段

#### 设计思路

保留当前的显式字段结构，通过代码生成工具自动生成模板代码。

```yaml
# middleware.yaml（中间件定义文件）
middlewares:
  - name: ratelimit
    type: RateLimitOptions
    fields:
      - name: QPS
        type: int
        default: 100
```

```bash
# 运行代码生成器
go run tools/gen-middleware/main.go middleware.yaml
# 自动生成：
# - pkg/options/middleware/ratelimit.go
# - 更新 options.go、merge.go
# - 更新所有服务的 options.go
```

#### 优缺点

| 维度 | 评分 | 说明 |
|------|------|------|
| **扩展性** | ⭐⭐⭐⭐ | 仍需手动运行生成器 |
| **类型安全** | ⭐⭐⭐⭐⭐ | 完全保留 |
| **IDE 支持** | ⭐⭐⭐⭐⭐ | 完全支持 |
| **YAML 支持** | ⭐⭐⭐⭐⭐ | 完全支持 |
| **可维护性** | ⭐⭐⭐ | 依赖生成工具，增加学习成本 |
| **实现难度** | ⭐⭐ | 需要开发和维护生成器工具 |

#### 工作量评估

- **修改文件数**：新增 1-2 个生成器工具
- **开发时间**：5-7 天
- **风险等级**：中（工具本身需要测试）

---

## 4. 方案推荐与实施路径

### 4.1 推荐方案：**方案 B（注册器模式 + 接口抽象）**

**推荐理由**：

1. **根本解决扩展性问题**
   新增中间件仅需修改 2-3 个位置，无需修改 `MergeSubOptions` 方法签名。

2. **平衡类型安全与灵活性**
   通过访问器方法保留类型安全，同时支持动态扩展。

3. **符合 Go 设计哲学**
   接口抽象 + 注册器是 Go 社区推崇的模式（如 `database/sql`、`net/http`）。

4. **渐进式迁移**
   可以逐步迁移现有中间件，不需要一次性重构。

5. **长期维护成本低**
   中间件自包含，符合单一职责原则，易于测试和维护。

### 4.2 实施路径（分 3 个阶段）

#### 阶段 1：基础设施建设（1-2 天）

**目标**：建立注册器和接口框架

**任务清单**：
- [ ] 定义 `MiddlewareConfig` 接口
- [ ] 实现 `Registry` 注册器
- [ ] 创建 `GetConfig()` 和类型安全访问器的生成模板
- [ ] 编写单元测试

**交付物**：
- `pkg/options/middleware/registry.go`
- `pkg/options/middleware/interface.go`

#### 阶段 2：迁移现有中间件（2-3 天）

**目标**：迁移现有 10 个中间件到新架构

**迁移顺序**（从简单到复杂）：
1. ✅ Health（最简单，无复杂依赖）
2. ✅ Metrics
3. ✅ Pprof
4. ✅ RequestID
5. ✅ Recovery
6. ✅ Logger
7. ✅ CORS
8. ✅ Timeout
9. ⚠️ Auth（有复杂依赖）
10. ⚠️ Authz（有复杂依赖）

**每个中间件的迁移步骤**：
- [ ] 实现 `MiddlewareConfig` 接口的所有方法
- [ ] 在 `init()` 中注册到全局注册器
- [ ] 添加类型安全访问器
- [ ] 更新单元测试

**交付物**：
- 所有中间件文件更新
- `pkg/options/middleware/options.go` 重构

#### 阶段 3：清理与优化（1 天）

**目标**：删除冗余代码，优化 API

**任务清单**：
- [ ] 删除 `MergeSubOptions()` 方法（或标记为废弃）
- [ ] 简化服务级 `options.go`（删除中间件字段）
- [ ] 更新文档和示例
- [ ] 性能测试和回归测试

**交付物**：
- 清理后的代码库
- 更新的文档（`docs/middleware-config.md`）

### 4.3 风险评估与应对

| 风险 | 等级 | 应对措施 |
|------|------|---------|
| **破坏现有 API** | 高 | 保留旧 API 并标记废弃，提供迁移指南 |
| **类型断言错误** | 中 | 在访问器中添加 `panic` 或返回 `error` |
| **YAML 解析兼容性** | 中 | 使用 `mapstructure` 的自定义解析钩子 |
| **性能回退** | 低 | 注册器使用 `map` 查找，性能影响可忽略 |
| **测试覆盖不足** | 中 | 要求每个中间件的单元测试覆盖率 > 80% |

### 4.4 回滚方案

如果实施过程中遇到无法解决的问题：

1. **保留旧 API**：`MergeSubOptions()` 方法在过渡期内保留
2. **双轨运行**：注册器和显式字段并存
3. **版本控制**：使用 Git 分支隔离变更
4. **回滚触发条件**：
   - 测试覆盖率下降 > 10%
   - 性能回退 > 5%
   - 迁移周期超过 1 周

---

## 5. 代码示例（方案 B）

### 5.1 核心接口定义

```go
// pkg/options/middleware/interface.go
package middleware

import "github.com/spf13/pflag"

// MiddlewareConfig 定义中间件配置的统一接口。
type MiddlewareConfig interface {
    // Validate 验证配置的有效性。
    Validate() error

    // Complete 完成配置的默认值填充。
    Complete() error

    // AddFlags 添加命令行标志。
    AddFlags(fs *pflag.FlagSet)

    // MergeFrom 从源配置合并（源配置优先）。
    MergeFrom(source MiddlewareConfig)
}
```

### 5.2 注册器实现

```go
// pkg/options/middleware/registry.go
package middleware

import (
    "fmt"
    "sync"
)

// Registry 中间件配置注册器。
type Registry struct {
    mu        sync.RWMutex
    factories map[string]func() MiddlewareConfig
}

var globalRegistry = &Registry{
    factories: make(map[string]func() MiddlewareConfig),
}

// Register 注册中间件配置工厂函数。
func Register(name string, factory func() MiddlewareConfig) {
    globalRegistry.mu.Lock()
    defer globalRegistry.mu.Unlock()

    if _, exists := globalRegistry.factories[name]; exists {
        panic(fmt.Sprintf("middleware %q already registered", name))
    }
    globalRegistry.factories[name] = factory
}

// Create 创建中间件配置实例。
func Create(name string) (MiddlewareConfig, error) {
    globalRegistry.mu.RLock()
    defer globalRegistry.mu.RUnlock()

    factory, ok := globalRegistry.factories[name]
    if !ok {
        return nil, fmt.Errorf("middleware %q not registered", name)
    }
    return factory(), nil
}

// ListRegistered 返回所有已注册的中间件名称。
func ListRegistered() []string {
    globalRegistry.mu.RLock()
    defer globalRegistry.mu.RUnlock()

    names := make([]string, 0, len(globalRegistry.factories))
    for name := range globalRegistry.factories {
        names = append(names, name)
    }
    return names
}
```

### 5.3 主配置结构重构

```go
// pkg/options/middleware/options.go
package middleware

import (
    "fmt"
    "github.com/spf13/pflag"
)

type Options struct {
    Enabled    []string
    configs    map[string]MiddlewareConfig  // 动态配置存储
    enabledSet map[string]bool
}

func NewOptions() *Options {
    return &Options{
        Enabled: append([]string{}, DefaultEnabledMiddlewares...),
        configs: make(map[string]MiddlewareConfig),
    }
}

// ensureDefaults 确保所有已注册的中间件配置已初始化。
func (o *Options) ensureDefaults() {
    for _, name := range ListRegistered() {
        if _, exists := o.configs[name]; !exists {
            cfg, err := Create(name)
            if err != nil {
                panic(err)  // 注册错误应在启动时暴露
            }
            o.configs[name] = cfg
        }
    }
}

// GetConfig 获取指定中间件的配置（通用方法）。
func (o *Options) GetConfig(name string) MiddlewareConfig {
    o.ensureDefaults()
    return o.configs[name]
}

// 类型安全的访问器（可通过代码生成）
func (o *Options) GetHealth() *HealthOptions {
    return o.GetConfig(MiddlewareHealth).(*HealthOptions)
}

func (o *Options) GetMetrics() *MetricsOptions {
    return o.GetConfig(MiddlewareMetrics).(*MetricsOptions)
}

// ... 其他访问器

// Validate 验证所有启用的中间件配置。
func (o *Options) Validate() error {
    o.ensureDefaults()
    o.buildEnabledSet()

    for _, name := range o.Enabled {
        if cfg, ok := o.configs[name]; ok {
            if err := cfg.Validate(); err != nil {
                return fmt.Errorf("middleware %q validation failed: %w", name, err)
            }
        } else {
            return fmt.Errorf("middleware %q not registered", name)
        }
    }
    return nil
}

// Complete 完成所有中间件配置。
func (o *Options) Complete() error {
    o.ensureDefaults()

    for name, cfg := range o.configs {
        if err := cfg.Complete(); err != nil {
            return fmt.Errorf("middleware %q complete failed: %w", name, err)
        }
    }
    return nil
}

// AddFlags 添加所有中间件的命令行标志。
func (o *Options) AddFlags(fs *pflag.FlagSet) {
    o.ensureDefaults()

    fs.StringSliceVar(&o.Enabled, "middleware.enabled", o.Enabled,
        fmt.Sprintf("List of enabled middlewares. Valid values: %v", ListRegistered()))

    for _, cfg := range o.configs {
        cfg.AddFlags(fs)
    }
}

// MergeAll 合并所有中间件配置（源配置优先）。
func (o *Options) MergeAll(sources map[string]MiddlewareConfig) {
    o.ensureDefaults()

    for name, source := range sources {
        if target, ok := o.configs[name]; ok {
            target.MergeFrom(source)
        }
    }
}
```

### 5.4 中间件实现示例（RateLimit）

```go
// pkg/options/middleware/ratelimit.go
package middleware

import (
    "errors"
    "github.com/spf13/pflag"
)

// 注册到全局注册器
func init() {
    Register(MiddlewareRateLimit, func() MiddlewareConfig {
        return NewRateLimitOptions()
    })
}

const MiddlewareRateLimit = "ratelimit"

type RateLimitOptions struct {
    QPS       int      `json:"qps" mapstructure:"qps"`
    Burst     int      `json:"burst" mapstructure:"burst"`
    SkipPaths []string `json:"skip-paths" mapstructure:"skip-paths"`
}

func NewRateLimitOptions() *RateLimitOptions {
    return &RateLimitOptions{
        QPS:       100,
        Burst:     200,
        SkipPaths: []string{},
    }
}

func (o *RateLimitOptions) Validate() error {
    if o.QPS <= 0 {
        return errors.New("QPS must be greater than 0")
    }
    if o.Burst < o.QPS {
        return errors.New("Burst must be >= QPS")
    }
    return nil
}

func (o *RateLimitOptions) Complete() error {
    return nil
}

func (o *RateLimitOptions) AddFlags(fs *pflag.FlagSet) {
    fs.IntVar(&o.QPS, "middleware.ratelimit.qps", o.QPS, "Requests per second")
    fs.IntVar(&o.Burst, "middleware.ratelimit.burst", o.Burst, "Burst size")
    fs.StringSliceVar(&o.SkipPaths, "middleware.ratelimit.skip-paths", o.SkipPaths, "Paths to skip rate limiting")
}

func (o *RateLimitOptions) MergeFrom(source MiddlewareConfig) {
    src, ok := source.(*RateLimitOptions)
    if !ok || src == nil {
        return
    }
    if src.QPS > 0 {
        o.QPS = src.QPS
    }
    if src.Burst > 0 {
        o.Burst = src.Burst
    }
    if len(src.SkipPaths) > 0 {
        o.SkipPaths = src.SkipPaths
    }
}
```

### 5.5 服务级配置简化

```go
// internal/api/options.go（重构后）
package app

import (
    middlewareopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
    // ... 其他导入
)

type Options struct {
    Server *serveropts.Options
    Log    *logopts.Options

    // 简化：只需一个 Middleware 字段，无需为每个中间件单独定义字段
    Middleware *middlewareopts.Options `json:"middleware" mapstructure:"middleware"`
}

func (o *Options) Complete() error {
    if o.Server != nil && o.Server.HTTP != nil {
        // 直接使用 Middleware 配置，无需 MergeSubOptions
        o.Server.HTTP.Middleware = o.Middleware
    }

    return o.Server.Complete()
}
```

---

## 6. 迁移检查清单

### 6.1 开发阶段

- [ ] 定义 `MiddlewareConfig` 接口
- [ ] 实现 `Registry` 注册器
- [ ] 重构 `Options` 主配置结构
- [ ] 迁移所有现有中间件（10 个）
- [ ] 添加类型安全访问器
- [ ] 更新单元测试（覆盖率 > 80%）

### 6.2 测试阶段

- [ ] 单元测试通过率 100%
- [ ] 集成测试验证配置加载
- [ ] 性能测试（对比基线）
- [ ] 兼容性测试（YAML 配置）
- [ ] 边界条件测试（未注册中间件、重复注册等）

### 6.3 文档阶段

- [ ] 更新 `docs/middleware-config.md`
- [ ] 添加迁移指南（`docs/migration-guide.md`）
- [ ] 更新示例配置文件
- [ ] 添加最佳实践文档

### 6.4 发布阶段

- [ ] 代码审查通过
- [ ] CI/CD 流水线绿灯
- [ ] 更新 CHANGELOG
- [ ] 发布 Release Notes

---

## 7. 参考资料

- [Practical Design Patterns in Go](https://leapcell.io/blog/practical-design-patterns-in-go-mastering-option-types-and-the-builder-pattern)
- [Go HTTP Middleware: Build Better APIs](https://dev.to/jones_charles_ad50858dbc0/go-http-middleware-build-better-apis-with-these-patterns-2nl2)
- [Middleware Patterns in Go](https://drstearns.github.io/tutorials/gomiddleware/)
- [5 API Design Patterns in Go (2025)](https://cristiancurteanu.com/5-api-design-patterns-in-go-that-solve-your-biggest-problems-2025/)
- [Handler and Middleware Design Pattern in Golang](https://medium.com/codex/handler-and-middleware-design-pattern-in-golang-de23ec452fce)

---

## 8. 总结

### 当前设计的核心问题

1. **扩展性差**：新增中间件需要修改 8-10 个文件
2. **方法签名污染**：`MergeSubOptions()` 参数列表持续增长
3. **配置重复**：每个服务重复定义中间件字段

### 推荐方案（方案 B）的核心优势

1. **修改点减少 70%**：从 8-10 个文件减少到 2-3 个
2. **符合开闭原则**：新增中间件无需修改现有代码
3. **保留类型安全**：通过访问器方法保证编译时检查
4. **长期维护成本低**：中间件自包含，易于测试和维护

### 实施建议

- **优先级**：高（解决架构级缺陷）
- **时间窗口**：1 周（分 3 个阶段实施）
- **风险等级**：中（需要充分测试）
- **投入产出比**：高（一次重构，长期受益）

---

**生成时间**：2026-01-03
**作者**：Claude Code（Backend Architect Agent）
