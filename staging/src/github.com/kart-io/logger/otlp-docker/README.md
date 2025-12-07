# OTLP Docker Stack

完整的OpenTelemetry日志、指标和追踪解决方案，支持以下流程：

```
应用程序 → OTEL Agent(4327) → OTEL Collector(4317) → VictoriaLogs(9428)
```

## 架构组件

### 核心组件

| 组件 | 端口 | 功能 | 健康检查 |
|------|------|------|----------|
| **OTEL Agent** | 4327 (gRPC), 4328 (HTTP) | 接收应用程序OTLP数据 | <http://localhost:13133> |
| **OTEL Collector** | 4317 (gRPC), 4318 (HTTP) | 数据处理和路由 | <http://localhost:13134> |
| **VictoriaLogs** | 9428 | 日志存储和查询 | <http://localhost:9428/health> |
| **Jaeger** | 16686 (UI), 14250 (gRPC) | 分布式追踪 | <http://localhost:16686> |
| **Prometheus** | 9090 | 指标存储 | <http://localhost:9090> |

### 数据流程

1. **应用程序** → 发送OTLP数据到Agent
2. **OTEL Agent** → 轻量级处理，转发到Collector
3. **OTEL Collector** → 复杂处理、过滤、路由
4. **后端存储** → VictoriaLogs(日志)、Jaeger(追踪)、Prometheus(指标)

## 快速开始

### 1. 部署服务

```bash
# 完整部署
./deploy.sh

# 仅清理旧资源
./deploy.sh --clean

# 跳过镜像拉取
./deploy.sh --no-pull
```

### 2. 测试链路

```bash
# 完整测试
./test.sh

# 仅检查服务状态
./test.sh --check-only

# 包含性能测试
./test.sh --perf
```

### 3. 停止服务

```bash
# 仅停止服务
./stop.sh

# 停止并删除数据
./stop.sh --volumes

# 完全清理
./stop.sh --all
```

## 应用程序集成

### Go应用程序示例

```go
package main

import (
    "time"

    "github.com/kart-io/logger"
    "github.com/kart-io/logger/option"
)

func main() {
    // 配置OTLP日志
    opt := &option.LogOption{
        Engine:      "zap",
        Level:       "INFO",
        Format:      "json",
        OutputPaths: []string{"stdout"},
        OTLP: &option.OTLPOption{
            Endpoint: "127.0.0.1:4327",  // 连接到Agent
            Protocol: "grpc",            // 或 "http"
            Timeout:  5 * time.Second,
        },
    }

    logger, err := logger.New(opt)
    if err != nil {
        panic(err)
    }

    // 发送结构化日志
    logger.Infow("用户登录",
        "user_id", 12345,
        "ip", "192.168.1.100",
        "timestamp", time.Now(),
    )
}
```

### 直连配置对比

| 连接方式 | 端点配置 | 优势 | 劣势 |
|----------|----------|------|------|
| **Agent链路** | `127.0.0.1:4327` | 完整的OTLP处理链路 | 稍高延迟 |
| **Collector直连** | `127.0.0.1:4317` | 跳过Agent处理 | 缺少Agent功能 |
| **VictoriaLogs直连** | `http://127.0.0.1:9428/insert/opentelemetry/v1/logs` | 最低延迟 | 仅支持日志 |

## 监控面板

### Web界面

- **VictoriaLogs查询**: <http://localhost:9428/select/logsql/query?query=>*
- **Jaeger追踪**: <http://localhost:16686>
- **Prometheus指标**: <http://localhost:9090>

### 指标端点

- **Agent指标**: <http://localhost:8888/metrics>
- **Collector指标**: <http://localhost:8889/metrics>
- **VictoriaLogs指标**: <http://localhost:9428/metrics>

## 配置文件说明

### OTEL Agent配置 (`otel-agent-config.yaml`)

- **轻量级处理**: 快速转发到Collector
- **资源限制**: 256MB内存限制
- **批处理**: 1秒超时，快速传输

### OTEL Collector配置 (`otel-collector-config.yaml`)

- **复杂处理**: 数据过滤、属性处理
- **多后端支持**: VictoriaLogs、Jaeger、Prometheus
- **资源限制**: 512MB内存限制
- **重要修复**: 正确的VictoriaLogs端点配置

## 故障排除

### 1. 服务状态检查

```bash
# 检查所有服务
docker-compose ps

# 检查特定服务日志
docker-compose logs otel-agent
docker-compose logs otel-collector
docker-compose logs victorialogs
```

### 2. 端口连通性测试

```bash
# 测试Agent端口
curl http://localhost:13133/
curl http://localhost:4327  # gRPC健康检查

# 测试VictoriaLogs
curl http://localhost:9428/health
```

### 3. 常见问题

| 问题 | 症状 | 解决方案 |
|------|------|----------|
| **路径重复错误** | `/insert/opentelemetry/v1/logs/v1/logs` | 使用新的Collector配置 |
| **连接超时** | OTLP export timeout | 检查网络和服务状态 |
| **日志丢失** | 发送成功但查询不到 | 检查VictoriaLogs日志格式 |
| **内存不足** | 容器OOM | 调整memory_limiter配置 |

### 4. 日志查询示例

```bash
# 查询所有日志
curl "http://localhost:9428/select/logsql/query?query=*&limit=10"

# 按级别查询
curl "http://localhost:9428/select/logsql/query?query=level:error"

# 按时间范围查询
curl "http://localhost:9428/select/logsql/query?query=_time:2025-08-29"

# 按字段查询
curl "http://localhost:9428/select/logsql/query?query=user_id:12345"
```

## 性能调优

### Agent优化

```yaml
processors:
  batch:
    timeout: 1s           # 快速转发
    send_batch_size: 1024 # 适中批次大小
```

### Collector优化

```yaml
processors:
  batch:
    timeout: 10s          # 较长批处理
    send_batch_size: 2048 # 较大批次
  memory_limiter:
    limit_mib: 512        # 足够的内存
```

## 生产环境建议

### 1. 安全配置

- 启用TLS加密
- 配置认证机制
- 限制网络访问

### 2. 高可用部署

- 多实例部署
- 负载均衡配置
- 数据备份策略

### 3. 监控告警

- 设置服务健康检查
- 配置指标告警
- 日志异常监控

## 版本信息

- **OTEL Collector**: 0.132.0
- **VictoriaLogs**: v1.28.0
- **Jaeger**: 1.57
- **Prometheus**: v2.51.0

## 支持

如有问题，请检查：

1. 运行 `./test.sh --check-only` 验证服务状态
2. 查看容器日志: `docker-compose logs [service]`
3. 检查端口占用: `netstat -tulpn | grep :4327`
4. 验证配置文件语法

---

**流程总结**: 应用程序 → OTEL Agent(4327) → OTEL Collector(4317) → VictoriaLogs(9428)
