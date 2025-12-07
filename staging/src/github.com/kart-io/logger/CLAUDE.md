# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

这是一个 Go 日志库项目 (`github.com/kart-io/logger`)，旨在提供智能 OTLP 配置、多源配置管理，以及跨 GORM、Kratos 等框架的统一日志记录。

**当前状态**：项目已初始化 Go module (Go 1.23.0)，核心接口和引擎实现已完成。项目包含完整的双引擎架构、字段标准化系统、配置管理和 OTLP 集成。提供了丰富的使用示例和测试代码。

**核心依赖**：项目基于 Go 标准库 `log/slog` 和高性能日志库 `go.uber.org/zap` 构建，提供统一的日志接口。

## 开发命令

Go 库项目的标准开发流程：

```bash
# 使用 Makefile（推荐）
make help              # 显示所有可用命令
make fmt               # 代码格式化和检查
make test              # 运行所有测试
make test-verbose      # 详细测试输出
make test-coverage     # 显示覆盖率
make coverage          # 生成覆盖率报告
make bench             # 运行基准测试（重要）
make build             # 构建库
make clean             # 清理构建产物
make check             # 格式化 + 测试

# 快速运行示例
make example-comprehensive
make example-performance
make example-otlp

# 手动命令
# 添加核心依赖（首次开发）
go get go.uber.org/zap
go get log/slog  # Go 1.21+ 标准库
go mod tidy

# 构建库
go build ./...

# 运行所有测试
go test ./...

# 运行特定测试
go test -run TestName ./path/to/package

# 运行基准测试（重要：日志库性能测试）
go test -bench=. ./...

# 运行测试并显示覆盖率
go test -cover ./...

# 运行测试（包含竞争检测，重要）
go test -race ./...

# 生成详细覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 代码格式化和静态检查
go fmt ./...
go vet ./...

# 检查 Go 版本兼容性
go mod tidy -go=1.23

# 运行示例（了解项目功能）
cd example/comprehensive && go run main.go
cd example/performance && go run main.go  
cd example/otlp && go run main.go
cd example/configuration && go run main.go
cd example/reload && go run main.go
cd example/rotation && go run main.go
cd example/integrations && go run main.go
cd example/echo && go run main.go           # Web服务器，可能端口冲突
cd example/initial_fields && go run main.go
cd example/zap && go run main.go

# 注意：一些子目录示例可能因为依赖问题无法独立运行
# 这是预期行为，因为它们依赖主项目的 go.mod

# 运行 OTLP 测试环境（需要 Docker）
cd otlp-docker && ./deploy.sh
```

## 项目架构

### 核心设计原则

基于需求文档 (`docs/REQUIREMENTS.md`)，日志器围绕以下关键架构概念设计：

1. **配置冲突解决**：多层优先级系统，处理环境变量、API 配置、配置中心和文件配置之间的冲突
2. **智能 OTLP 检测**：基于端点配置自动启用 OTLP，消除冗余的 `enabled: true` 设置
3. **扁平化配置**：顶级字段（如 `otlp-endpoint`）用于常见场景，嵌套配置用于高级控制
4. **动态配置**：通过 HTTP API、信号和配置中心集成进行运行时配置变更
5. **智能输出控制**：基于启用功能的日志智能路由到控制台、文件和 OTLP

### 配置优先级顺序（从高到低）

1. 环境变量（启动时读取一次，运行时变更需要重载机制）
2. HTTP API 配置（运行时动态）
3. 配置中心（Nacos/Consul/etcd）
4. 配置文件（YAML）
5. 系统默认值

### 技术架构

项目采用双日志引擎架构，通过统一接口层实现引擎抽象：

- **Zap 引擎**：高性能结构化日志，用于生产环境和性能敏感场景
- **Slog 引擎**：Go 1.21+ 标准库，提供标准化接口和兼容性
- **统一接口层**：`Logger` 接口定义了标准的日志方法，支持三种调用风格：
  - 基础方法：`Debug(args...)`, `Info(args...)` 等
  - 格式化方法：`Debugf(template, args...)`, `Infof(template, args...)` 等
  - 结构化方法：`Debugw(msg, keyValues...)`, `Infow(msg, keyValues...)` 等
- **统一字段标准化**：确保不同底层库（Zap/Slog）输出完全一致的字段名和格式
- **增强功能**：支持上下文注入 (`WithCtx`)、字段追加 (`With`)、调用栈跳过 (`WithCallerSkip`) 等

### 核心包结构

基于需求文档、双引擎架构和统一字段标准化要求，项目采用以下包组织：

- **`core/`**: 核心接口定义，包含 `Logger` 接口和 `Level` 类型
- **`fields/`**: 统一字段定义包，标准化所有日志字段名称和格式
- **`engines/zap/`**: Zap 日志引擎实现，严格遵循统一字段标准
- **`engines/slog/`**: Slog 日志引擎实现，严格遵循统一字段标准
- **`factory/`**: 日志器工厂包，处理多源配置冲突和优先级，创建日志器实例
- **`otlp/`**: OTLP 集成包，智能检测和自动配置
- **`option/`**: 配置选项包，提供详细的配置验证和标志绑定
- **`reload/`**: 配置重载包，支持运行时动态配置更新
- **`runtime/`**: 运行时管理包，处理生命周期和状态管理
- **`errors/`**: 错误处理包，提供统一的错误日志记录策略
- **`integrations/`**: 框架集成包，包含 GORM、Kratos、Gin 等适配器
- **`example/`**: 完整的使用示例，包含性能测试和 OTLP 集成演示
- **`cmd/demo/`**: 命令行演示程序
- **`otlp-docker/`**: OTLP 测试环境的 Docker 配置

### 关键组件

- **Logger 接口**：统一的日志接口，支持三种调用风格和增强功能
- **字段标准化系统**：定义统一的字段名称、格式和输出规范，确保不同引擎输出一致
- **引擎实现**：Zap 和 Slog 引擎分别实现 Logger 接口，严格遵循字段标准
- **工厂模式**：根据配置动态创建和切换日志引擎
- **配置冲突处理器**：解决多源配置冲突，优先考虑明确的用户意图
- **OTLP 自动检测器**：基于端点可用性的智能启用
- **动态配置重载器**：处理通过信号/API 的运行时配置变更
- **框架适配器**：GORM、Kratos 和其他框架的统一日志记录
- **多源配置管理器**：管理来自文件、环境、API 和配置中心的配置
- **回滚机制**：配置变更的多级智能回滚

### 字段标准化要求

**核心原则**：无论底层使用 Zap 还是 Slog，输出的日志字段必须完全一致：

- **时间字段**：统一使用 `timestamp` 或 `time` 字段名
- **级别字段**：统一使用 `level` 字段名，值格式一致（如 `DEBUG`, `INFO` 等）
- **消息字段**：统一使用 `message` 或 `msg` 字段名
- **调用者字段**：统一使用 `caller` 字段名和格式
- **跟踪字段**：统一使用 `trace_id`, `span_id` 等字段名
- **自定义字段**：用户通过 `With()` 和 `Debugw()` 等方法添加的字段保持原样

### 状态术语

项目使用特定术语来区分配置状态：

- **禁用 (disabled)**：用户明确设置 `enabled: false`
- **未启用 (not enabled)**：由于缺少配置而未启用功能
- **自动禁用 (auto disabled)**：系统智能判断应禁用功能

## 需求文档

详细需求记录在 `docs/REQUIREMENTS.md`（中文）中，涵盖：

- 核心用户痛点和目标用户
- 配置冲突解决算法
- OTLP 智能检测逻辑
- 动态配置和回滚机制
- 框架集成策略
- 多环境配置管理

实现功能时应参考此文档，确保与指定行为和边界情况处理保持一致。

## 开发注意事项

- 项目使用 Apache 2.0 许可证
- 需求文档为中文，实现应支持中英文日志记录
- **性能要求**：作为日志库，性能至关重要，必须进行基准测试
- **接口设计**：严格遵循 `Logger` 接口定义，确保引擎实现的一致性
- **字段标准化**：这是项目的核心要求，必须确保不同引擎输出完全一致的字段格式
- **引擎切换透明性**：用户可以在 Zap 和 Slog 之间切换而不影响日志输出格式
- **兼容性**：同时支持 Zap 和 Slog，确保平滑迁移路径
- **零分配设计**：在热路径上避免内存分配，特别是 Zap 引擎
- **调用风格支持**：同时支持简单参数、格式化字符串和结构化键值对三种风格
- 专注于开发者体验 - 最小化配置复杂性，同时提供高级控制
- 需求中规定了大量配置冲突的边界情况处理
- 项目强调智能默认值和自动配置以减少设置摩擦

## 规格驱动开发

本项目遵循规格驱动开发方法，采用结构化的需求 → 设计 → 实现工作流程。使用 Kiro 规格工具 (`/kiro:spec`) 来：

- 从需求创建功能规格
- 生成具有适当架构的设计文档
- 将功能分解为可操作的实现任务

`docs/REQUIREMENTS.md` 中的全面需求是所有功能开发的基础。

## 中文开发环境适配

- 需求文档和注释优先使用中文
- 日志输出支持中文字符集
- 错误消息和状态描述提供中文版本
- 配置字段支持中文说明和示例

## 示例和测试

项目提供了丰富的示例代码，展示所有功能：

- **`example/comprehensive/`**: 完整的功能演示，包含所有 15 个日志方法
- **`example/performance/`**: 性能基准测试，比较 Slog 和 Zap 引擎
- **`example/otlp/`**: OTLP 集成测试，包含多种配置方式
- **`example/configuration/`**: 配置管理演示，展示多源配置处理
- **`otlp-docker/`**: 完整的 OTLP 测试环境，包含 Docker Compose 配置

## 核心文件路径

关键文件位置，便于快速定位：

- **主入口**: `logger.go` - 全局日志器和包级便利函数
- **核心接口**: `core/logger.go` - Logger 接口定义
- **工厂实现**: `factory/factory.go` - 日志器创建逻辑
- **配置选项**: `option/option.go` - 配置结构和验证
- **Zap 引擎**: `engines/zap/zap.go` - Zap 实现
- **Slog 引擎**: `engines/slog/slog.go` - Slog 实现
- **字段标准**: `fields/fields.go` - 统一字段定义
- **配置重载**: `reload/reloader.go` - 运行时配置重载
- **错误处理**: `errors/handler.go` - 统一错误日志策略
- **OTLP 集成**: `otlp/provider.go` - OpenTelemetry 提供者
- **需求文档**: `docs/REQUIREMENTS.md` - 详细的中文需求文档

## 测试和验证

运行特定测试的命令：

```bash
# 测试特定包
go test ./core
go test ./engines/zap
go test ./engines/slog
go test ./factory
go test ./reload
go test ./errors
go test ./integrations/gorm
go test ./integrations/kratos
go test ./integrations/gin

# 测试特定功能
go test -run TestLoggerFactory ./factory
go test -run TestOTLP ./otlp
go test -run TestReloader ./reload
go test -run TestErrorHandler ./errors

# 性能测试
go test -bench=BenchmarkZap ./engines/zap
go test -bench=BenchmarkSlog ./engines/slog

# 运行完整基准测试套件
cd example/performance && go run main.go

# 重要：竞争检测（项目已修复所有已知数据竞争）
go test -race ./...
```

### 已知问题和解决方案

项目已修复以下关键问题：

1. **数据竞争修复**：
   - `reload/reloader_test.go`: 添加互斥锁保护共享变量
   - `errors/handler_test.go`: 添加互斥锁保护回调状态

2. **测试环境优化**：
   - Zap 引擎：处理 stdout/stderr sync 错误
   - Slog 引擎：处理文件关闭错误
   - 这些错误在实际生产环境中不会出现

3. **示例程序状态**：
   - 所有主目录示例均可正常运行
   - 子目录示例因依赖隔离可能无法独立运行（预期行为）

## 快捷备忘录

使用 `# <重要信息>` 格式添加到此处：

# 关键模块: core.Logger 接口是所有引擎的统一入口点
# 字段标准: 所有引擎必须输出完全一致的字段格式
# 性能目标: Zap 引擎零分配，Slog 引擎最小分配
# 配置约定: 支持扁平化(otlp-endpoint)和嵌套(otlp.endpoint)两种配置风格
# OTLP 逻辑: 有端点即启用，无需显式 enabled: true
# 热重载: 支持信号驱动和 API 驱动的配置重载机制
# 框架集成: 提供 GORM、Kratos、Gin 等主流框架适配器
# 错误恢复: 配置重载失败时自动回滚到上一个稳定配置
# 测试要求: 必须运行 go test -race 检测数据竞争，项目已修复所有已知竞争条件
# 示例状态: 主要示例都可正常运行，部分子目录示例因依赖隔离无法独立运行（预期行为）
# 日志轮转: 支持文件大小和时间基础的日志轮转，配置在 rotation 示例中
# Flush 处理: 引擎已优化处理测试环境的 stdout/stderr sync 错误

<!--
快速添加格式示例：
# 新依赖: go get github.com/example/package
# 配置约定: 所有配置项必须支持环境变量覆盖
# 性能要求: 日志操作不能超过 100ns
# 字段标准: 时间字段统一使用 "timestamp"
-->
