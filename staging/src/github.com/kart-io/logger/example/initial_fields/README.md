# InitialFields 使用示例

这个示例演示了如何使用 kart-io/logger 的 InitialFields 功能来自动为所有日志条目添加通用字段。

## 🎯 功能演示

- ✅ **WithInitialFields**: 批量添加初始字段
- ✅ **AddInitialField**: 逐个添加字段（支持链式调用）
- ✅ **GetInitialFields**: 获取已配置的字段
- ✅ **字段优先级**: 展示字段覆盖规则
- ✅ **多 Logger 实例**: 不同服务的独立配置

## 🚀 运行示例

```bash
# 进入示例目录
cd example/initial_fields

# 运行示例
go run main.go
```

## 📋 示例输出

运行示例后，你将看到：

### 1. 配置的初始字段
```
=== InitialFields Usage Example ===

Configured InitialFields:
  service.version: v1.0.0
  environment: development
  datacenter: local
  instance_id: example-001
  team: platform
  project: logging-demo
  build_id: 12345
  service.name: example-service
```

### 2. 自动包含的字段
每个日志条目都会自动包含所有初始字段：

```json
{
  "time": "2025-09-01T18:59:52.659628186+08:00",
  "level": "info",
  "msg": "Application started",
  "team": "platform",
  "project": "logging-demo",
  "build_id": "12345",
  "environment": "development",
  "datacenter": "local",
  "service.name": "example-service",
  "service.version": "v1.0.0",
  "instance_id": "example-001",
  "engine": "slog",
  "caller": "initial_fields/main.go:51"
}
```

### 3. 字段优先级演示
```json
{
  "time": "2025-09-01T18:59:52.659677026+08:00",
  "level": "info",
  "msg": "Field precedence test",
  "service.name": "example-service",        // InitialFields 值
  "service.name": "overridden-by-with",     // With() 方法覆盖
  "service.version": "overridden-by-current-call", // 当前调用最高优先级
  "additional_field": "only-in-this-log"
}
```

## 🔍 代码亮点

### 三种添加方法
```go
// 方法1: 批量添加
opt.WithInitialFields(map[string]interface{}{
    "service.name":    "example-service",
    "service.version": "v1.0.0",
})

// 方法2: 链式调用
opt.AddInitialField("environment", "development").
    AddInitialField("datacenter", "local").
    AddInitialField("instance_id", "example-001")

// 方法3: 混合使用
opt.WithInitialFields(map[string]interface{}{
    "team":    "platform",
    "project": "logging-demo",
}).AddInitialField("build_id", "12345")
```

### 字段优先级规则
```go
// 优先级: 当前调用 > With() > InitialFields
childLogger := serviceLogger.With("service.name", "overridden-by-with")
childLogger.Infow("Field precedence test",
    "service.version", "overridden-by-current-call", // 最高优先级
    "additional_field", "only-in-this-log",
)
```

### 不同服务实例
```go
// 为不同服务创建独立的日志器
opt2 := option.DefaultLogOption().
    AddInitialField("service.name", "another-service").
    AddInitialField("service.version", "v2.0.0").
    AddInitialField("component", "worker")

logger2, _ := logger.New(opt2)
```

## 🎪 实际应用场景

### 1. 微服务环境
每个微服务实例自动包含服务标识信息：
- `service.name` - 服务名称
- `service.version` - 服务版本
- `instance_id` - 实例标识
- `environment` - 部署环境

### 2. 分布式追踪
与 OTLP 集成，确保所有日志都包含服务标识，便于在 Jaeger、VictoriaLogs 等后端进行追踪和分析。

### 3. 多租户应用
为不同租户、项目或团队添加标识字段，便于日志聚合和权限控制。

## 📚 相关文档

- [📘 Option Package](../../option/README.md) - 详细的 InitialFields API 文档
- [📘 Main README](../../README.md) - 完整的 logger 使用指南
- [📡 OTLP Example](../otlp/) - OTLP 集成中如何使用 InitialFields

## 🔗 相关示例

- [📋 Comprehensive](../comprehensive/) - 完整功能演示
- [🔧 Configuration](../configuration/) - 配置管理
- [📡 OTLP](../otlp/) - OpenTelemetry 集成

---

这个示例展示了 InitialFields 的强大功能，让你的日志更加结构化和易于管理！ 🚀