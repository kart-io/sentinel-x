# 生产环境日志管理最佳实践

## 概述

本文档提供了在生产环境中使用 kart-io/logger 的最佳实践，特别关注日志轮转、性能优化、监控和故障排除。

## 架构设计原则

### 1. 分层日志策略

```
应用层日志     → 业务逻辑、用户行为
├── 访问日志   → HTTP请求、API调用  (7天保留)
├── 业务日志   → 核心业务流程      (15天保留) 
├── 错误日志   → 异常和错误       (30天保留)
└── 审计日志   → 安全和合规       (12个月保留)

系统层日志     → 基础设施、性能监控
├── 性能日志   → 响应时间、资源使用 (7天保留)
├── 调试日志   → 开发调试信息     (3天保留)
└── 运维日志   → 部署、配置变更    (30天保留)
```

### 2. 日志级别使用规范

| 级别 | 用途 | 生产环境建议 | 示例场景 |
|------|------|-------------|----------|
| DEBUG | 详细调试信息 | 关闭 | 变量值、执行路径 |
| INFO | 正常业务流程 | 开启 | 用户登录、订单创建 |
| WARN | 潜在问题预警 | 开启 | 性能阈值、降级处理 |
| ERROR | 错误但可恢复 | 开启 | 业务异常、重试失败 |
| FATAL | 系统致命错误 | 开启 | 数据库连接失败 |

## 轮转策略配置

### 高并发应用（日PV > 100万）

```go
// 按小时轮转，严格控制单文件大小
func createHighVolumeRotation() *lumberjack.Logger {
    return &lumberjack.Logger{
        Filename:   "/var/log/app/app.log",
        MaxSize:    50,    // 50MB 快速轮转
        MaxBackups: 72,    // 3天 * 24小时
        MaxAge:     3,     // 3天清理
        Compress:   true,  // 必须压缩
        LocalTime:  true,
    }
}
```

### 中等流量应用（日PV 1万-100万）

```go
// 每日轮转，平衡存储和查询需求
func createMediumVolumeRotation() *lumberjack.Logger {
    return &lumberjack.Logger{
        Filename:   "/var/log/app/app.log", 
        MaxSize:    200,   // 200MB
        MaxBackups: 15,    // 15天
        MaxAge:     15,    // 15天清理
        Compress:   true,
        LocalTime:  true,
    }
}
```

### 低流量应用（日PV < 1万）

```go
// 按周轮转，长期保留
func createLowVolumeRotation() *lumberjack.Logger {
    return &lumberjack.Logger{
        Filename:   "/var/log/app/app.log",
        MaxSize:    500,   // 500MB
        MaxBackups: 12,    // 12周
        MaxAge:     90,    // 90天清理
        Compress:   true,
        LocalTime:  true,
    }
}
```

## 分类日志管理

### 1. 访问日志配置

```go
// 高频访问日志：快速轮转，短期保留
func createAccessLogRotation() *lumberjack.Logger {
    return &lumberjack.Logger{
        Filename:   "/var/log/app/access.log",
        MaxSize:    100,   // 100MB
        MaxBackups: 14,    // 7天（每天2个文件）
        MaxAge:     7,     // 7天清理
        Compress:   true,
        LocalTime:  true,
    }
}

// 访问日志专用 logger
func createAccessLogger() core.Logger {
    writer := createAccessLogRotation()
    logger := NewRotationLogger(writer, core.InfoLevel, "json")
    
    return logger.With(
        "log_type", "access",
        "component", "http-server",
    )
}
```

### 2. 错误日志配置

```go
// 错误日志：长期保留，便于故障分析
func createErrorLogRotation() *lumberjack.Logger {
    return &lumberjack.Logger{
        Filename:   "/var/log/app/error.log",
        MaxSize:    50,    // 较小文件便于分析
        MaxBackups: 60,    // 30天
        MaxAge:     30,    // 30天清理
        Compress:   true,
        LocalTime:  true,
    }
}
```

### 3. 审计日志配置

```go
// 审计日志：最高保留级别
func createAuditLogRotation() *lumberjack.Logger {
    return &lumberjack.Logger{
        Filename:   "/var/log/app/audit.log",
        MaxSize:    100,
        MaxBackups: 365,   // 365天
        MaxAge:     365,   // 1年
        Compress:   false, // 审计日志不压缩
        LocalTime:  true,
    }
}
```

## 性能优化

### 1. 缓冲区优化

```go
// Zap 引擎优化配置
func createOptimizedZapLogger() core.Logger {
    // 使用缓冲写入器
    writer := zapcore.AddSync(&lumberjack.Logger{
        Filename:   "/var/log/app/app.log",
        MaxSize:    200,
        MaxBackups: 15,
        MaxAge:     15,
        Compress:   true,
        LocalTime:  true,
    })

    // 配置编码器以优化性能
    encoderConfig := zap.NewProductionEncoderConfig()
    encoderConfig.TimeKey = "timestamp"
    encoderConfig.EncodeTime = zapcore.EpochNanoTimeEncoder // 更快的时间编码
    
    encoder := zapcore.NewJSONEncoder(encoderConfig)
    core := zapcore.NewCore(encoder, writer, zapcore.InfoLevel)
    
    // 添加缓冲
    bufferedCore := zapcore.NewBufferedWriteSyncer(writer, 256*1024) // 256KB 缓冲区
    
    return &ZapLogger{
        zap: zap.New(core, zap.AddCaller()),
    }
}
```

### 2. 异步日志

```go
// 异步日志写入器
type AsyncLogger struct {
    core.Logger
    buffer chan logEntry
    done   chan struct{}
}

type logEntry struct {
    level   core.Level
    message string
    fields  []interface{}
}

func NewAsyncLogger(underlying core.Logger, bufferSize int) *AsyncLogger {
    al := &AsyncLogger{
        Logger: underlying,
        buffer: make(chan logEntry, bufferSize),
        done:   make(chan struct{}),
    }
    
    go al.processLogs()
    return al
}

func (al *AsyncLogger) processLogs() {
    for {
        select {
        case entry := <-al.buffer:
            switch entry.level {
            case core.InfoLevel:
                al.Logger.Infow(entry.message, entry.fields...)
            case core.ErrorLevel:
                al.Logger.Errorw(entry.message, entry.fields...)
            // ... 其他级别
            }
        case <-al.done:
            return
        }
    }
}

func (al *AsyncLogger) Infow(msg string, keysAndValues ...interface{}) {
    select {
    case al.buffer <- logEntry{core.InfoLevel, msg, keysAndValues}:
    default:
        // 缓冲区满时降级到同步写入
        al.Logger.Infow(msg, keysAndValues...)
    }
}
```

## 监控和告警

### 1. 日志监控指标

```go
// 日志监控器
type LogMonitor struct {
    logger       core.Logger
    metrics      *prometheus.CounterVec
    errorRate    *prometheus.GaugeVec
    lastRotation time.Time
}

func NewLogMonitor(logger core.Logger) *LogMonitor {
    metrics := prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "log_entries_total",
            Help: "Total number of log entries by level",
        },
        []string{"level", "service"},
    )
    
    errorRate := prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "log_error_rate",
            Help: "Error log rate per minute",
        },
        []string{"service"},
    )
    
    prometheus.MustRegister(metrics, errorRate)
    
    return &LogMonitor{
        logger:    logger,
        metrics:   metrics,
        errorRate: errorRate,
    }
}

func (lm *LogMonitor) Infow(msg string, keysAndValues ...interface{}) {
    lm.metrics.WithLabelValues("info", "myapp").Inc()
    lm.logger.Infow(msg, keysAndValues...)
}

func (lm *LogMonitor) Errorw(msg string, keysAndValues ...interface{}) {
    lm.metrics.WithLabelValues("error", "myapp").Inc()
    lm.errorRate.WithLabelValues("myapp").Inc()
    lm.logger.Errorw(msg, keysAndValues...)
}
```

### 2. 告警规则

```yaml
# Prometheus 告警规则
groups:
  - name: log_alerts
    rules:
    - alert: HighErrorRate
      expr: rate(log_entries_total{level="error"}[5m]) > 10
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: "High error log rate detected"
        description: "Error rate is {{ $value }} per second"
        
    - alert: LogRotationFailed
      expr: time() - log_last_rotation_timestamp > 86400
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "Log rotation has not occurred in 24 hours"
        
    - alert: DiskSpaceHigh
      expr: (node_filesystem_size_bytes{mountpoint="/var/log"} - node_filesystem_free_bytes{mountpoint="/var/log"}) / node_filesystem_size_bytes{mountpoint="/var/log"} > 0.8
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Log directory disk usage is high"
```

## 安全和合规

### 1. 敏感信息过滤

```go
// 敏感信息过滤器
type SensitiveFilter struct {
    core.Logger
    filters []FilterRule
}

type FilterRule struct {
    Field    string
    Pattern  *regexp.Regexp
    Replace  string
}

func NewSensitiveFilter(logger core.Logger) *SensitiveFilter {
    filters := []FilterRule{
        {
            Field:   "password",
            Pattern: regexp.MustCompile(`.*`),
            Replace: "***",
        },
        {
            Field:   "credit_card",
            Pattern: regexp.MustCompile(`\d{4}\d{4}\d{4}\d{4}`),
            Replace: "****-****-****-$1",
        },
        {
            Field:   "phone",
            Pattern: regexp.MustCompile(`(\d{3})\d{4}(\d{4})`),
            Replace: "$1****$2",
        },
    }
    
    return &SensitiveFilter{
        Logger:  logger,
        filters: filters,
    }
}

func (sf *SensitiveFilter) Infow(msg string, keysAndValues ...interface{}) {
    filtered := sf.filterSensitiveData(keysAndValues)
    sf.Logger.Infow(msg, filtered...)
}

func (sf *SensitiveFilter) filterSensitiveData(keysAndValues []interface{}) []interface{} {
    result := make([]interface{}, len(keysAndValues))
    copy(result, keysAndValues)
    
    for i := 0; i < len(result); i += 2 {
        if i+1 < len(result) {
            key := fmt.Sprintf("%v", result[i])
            value := fmt.Sprintf("%v", result[i+1])
            
            for _, filter := range sf.filters {
                if strings.Contains(strings.ToLower(key), filter.Field) {
                    result[i+1] = filter.Pattern.ReplaceAllString(value, filter.Replace)
                    break
                }
            }
        }
    }
    
    return result
}
```

### 2. 访问控制

```bash
# 日志文件权限设置
chmod 640 /var/log/app/*.log    # 应用可读写，组可读
chown app:log-readers /var/log/app/*.log

# 审计日志特殊权限
chmod 600 /var/log/app/audit.log  # 仅应用可访问
chown app:audit /var/log/app/audit.log
```

## 故障排除

### 1. 常见问题诊断

```bash
# 检查日志轮转状态
/usr/local/bin/myapp-log-rotate status

# 检查磁盘空间
df -h /var/log/

# 检查日志写入权限
sudo -u myapp touch /var/log/myapp/test.log && rm /var/log/myapp/test.log

# 检查 logrotate 配置
logrotate -d /etc/logrotate.d/myapp

# 查看 logrotate 执行历史
grep myapp /var/log/logrotate.log
```

### 2. 应急处理脚本

```bash
#!/bin/bash
# 日志应急清理脚本

APP_NAME="myapp"
LOG_DIR="/var/log/$APP_NAME"
EMERGENCY_THRESHOLD="90"  # 磁盘使用率阈值

# 获取磁盘使用率
disk_usage=$(df "$LOG_DIR" | awk 'NR==2 {gsub(/%/, "", $5); print $5}')

if [ "$disk_usage" -gt "$EMERGENCY_THRESHOLD" ]; then
    echo "紧急情况：磁盘使用率 ${disk_usage}%，开始清理..."
    
    # 1. 强制轮转当前日志
    logrotate -f "/etc/logrotate.d/$APP_NAME"
    
    # 2. 删除最旧的压缩文件
    find "$LOG_DIR" -name "*.gz" -type f -mtime +7 -delete
    
    # 3. 清理临时文件
    find "$LOG_DIR" -name "*.tmp" -type f -delete
    
    # 4. 压缩大文件
    find "$LOG_DIR" -name "*.log.*" -size +100M ! -name "*.gz" -exec gzip {} \;
    
    echo "清理完成，当前磁盘使用率："
    df -h "$LOG_DIR"
fi
```

## 容器化部署

### 1. Docker 配置

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o myapp

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata logrotate
WORKDIR /root/

# 创建日志目录
RUN mkdir -p /var/log/myapp
COPY --from=builder /app/myapp .

# 复制 logrotate 配置
COPY configs/logrotate.conf /etc/logrotate.d/myapp

# 启动脚本
COPY scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080
VOLUME ["/var/log/myapp"]

ENTRYPOINT ["/entrypoint.sh"]
CMD ["./myapp"]
```

### 2. Kubernetes 配置

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: myapp:latest
        env:
        - name: LOG_LEVEL
          value: "INFO"
        - name: LOG_OUTPUT_PATHS
          value: "/var/log/myapp/app.log"
        volumeMounts:
        - name: log-volume
          mountPath: /var/log/myapp
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      - name: log-shipper
        image: fluent/fluent-bit:latest
        volumeMounts:
        - name: log-volume
          mountPath: /var/log/myapp
        - name: fluentbit-config
          mountPath: /fluent-bit/etc
      volumes:
      - name: log-volume
        emptyDir: {}
      - name: fluentbit-config
        configMap:
          name: fluentbit-config
```

## 监控仪表板

### 1. Grafana 仪表板配置

```json
{
  "dashboard": {
    "title": "Application Logs Dashboard",
    "panels": [
      {
        "title": "Log Volume by Level",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(log_entries_total[5m])",
            "legendFormat": "{{ level }}"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "singlestat",
        "targets": [
          {
            "expr": "rate(log_entries_total{level=\"error\"}[5m])"
          }
        ]
      },
      {
        "title": "Log File Sizes",
        "type": "table",
        "targets": [
          {
            "expr": "node_filesystem_size_bytes{mountpoint=\"/var/log\"}"
          }
        ]
      }
    ]
  }
}
```

## 总结

### 关键要点

1. **分层管理**：根据日志类型制定不同的保留策略
2. **性能优化**：使用缓冲区和异步写入减少 I/O 影响
3. **安全合规**：过滤敏感信息，设置适当的文件权限
4. **监控告警**：实时监控日志质量和系统健康状态
5. **故障处理**：准备应急清理和恢复脚本

### 部署检查清单

- [ ] 日志目录权限配置正确
- [ ] logrotate 配置已测试
- [ ] 监控指标已配置
- [ ] 告警规则已设置
- [ ] 应急脚本已准备
- [ ] 敏感信息过滤已启用
- [ ] 容器化配置已验证
- [ ] 备份策略已实施

遵循这些最佳实践，可以确保 kart-io/logger 在生产环境中稳定、高效、安全地运行。