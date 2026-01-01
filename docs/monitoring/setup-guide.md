# Sentinel-X RAG 服务监控设置指南

本指南说明如何为 Sentinel-X RAG 服务搭建完整的监控体系。

## 概述

监控架构包含以下组件：
- **Prometheus**: 指标收集和存储
- **Grafana**: 可视化仪表盘
- **Alertmanager**: 告警管理和路由

## 快速开始

### 1. 启动 Prometheus

#### Docker Compose 方式

创建 `docker-compose.monitoring.yml`:

```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: sentinel-x-prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - ./monitoring/rules:/etc/prometheus/rules
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/usr/share/prometheus/console_libraries'
      - '--web.console.templates=/usr/share/prometheus/consoles'
      - '--web.enable-lifecycle'
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:latest
    container_name: sentinel-x-grafana
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana/provisioning:/etc/grafana/provisioning
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    networks:
      - monitoring
    depends_on:
      - prometheus

  alertmanager:
    image: prom/alertmanager:latest
    container_name: sentinel-x-alertmanager
    ports:
      - "9093:9093"
    volumes:
      - ./monitoring/alertmanager.yml:/etc/alertmanager/alertmanager.yml
      - alertmanager_data:/alertmanager
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
      - '--storage.path=/alertmanager'
    networks:
      - monitoring

volumes:
  prometheus_data:
  grafana_data:
  alertmanager_data:

networks:
  monitoring:
    driver: bridge
```

启动监控栈：

```bash
docker-compose -f docker-compose.monitoring.yml up -d
```

### 2. 配置 Prometheus

创建 `monitoring/prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'sentinel-x'
    environment: 'production'

# 告警管理器配置
alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - alertmanager:9093

# 告警规则文件
rule_files:
  - '/etc/prometheus/rules/*.yml'

# 抓取配置
scrape_configs:
  # RAG 服务指标
  - job_name: 'rag-service'
    metrics_path: '/v1/rag/metrics'
    static_configs:
      - targets:
          - 'host.docker.internal:8081'  # RAG 服务地址
        labels:
          service: 'rag'
          component: 'backend'
    scrape_interval: 15s
    scrape_timeout: 10s

  # Prometheus 自身指标
  - job_name: 'prometheus'
    static_configs:
      - targets:
          - 'localhost:9090'
```

创建告警规则文件 `monitoring/rules/rag-alerts.yml`（内容参见 `alerting-rules.md`）。

### 3. 配置 Grafana

#### 3.1 数据源配置

创建 `monitoring/grafana/provisioning/datasources/prometheus.yml`:

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
```

#### 3.2 仪表盘配置

创建 `monitoring/grafana/provisioning/dashboards/dashboards.yml`:

```yaml
apiVersion: 1

providers:
  - name: 'Sentinel-X'
    orgId: 1
    folder: 'RAG Service'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
```

将 `docs/monitoring/grafana-dashboard.json` 复制到 `monitoring/grafana/provisioning/dashboards/` 目录。

### 4. 配置 Alertmanager

创建 `monitoring/alertmanager.yml`:

```yaml
global:
  resolve_timeout: 5m
  slack_api_url: 'YOUR_SLACK_WEBHOOK_URL'

# 告警路由
route:
  receiver: 'default'
  group_by: ['alertname', 'component', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h

  routes:
    # Critical 告警路由
    - match:
        severity: critical
      receiver: 'slack-critical'
      continue: true

    # Warning 告警路由
    - match:
        severity: warning
      receiver: 'slack-warnings'

    # 成本告警特殊路由
    - match:
        type: cost
      receiver: 'slack-cost'

# 接收器配置
receivers:
  - name: 'default'
    slack_configs:
      - channel: '#rag-monitoring'
        title: '{{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

  - name: 'slack-critical'
    slack_configs:
      - channel: '#rag-alerts-critical'
        title: '🔴 CRITICAL: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
        send_resolved: true

  - name: 'slack-warnings'
    slack_configs:
      - channel: '#rag-alerts-warning'
        title: '⚠️  WARNING: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

  - name: 'slack-cost'
    slack_configs:
      - channel: '#cost-alerts'
        title: '💰 Cost Alert: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'

# 告警抑制规则
inhibit_rules:
  - source_match:
      alertname: RAGCircuitBreakerOpen
    target_match:
      alertname: RAGLLMHighErrorRate
    equal: ['component', 'service']
```

## 验证监控系统

### 1. 验证 Prometheus 指标采集

```bash
# 检查 Prometheus targets 状态
curl http://localhost:9090/api/v1/targets

# 查询指标
curl http://localhost:9090/api/v1/query?query=sentinel_x_rag_queries_total
```

### 2. 验证 RAG 服务指标导出

```bash
# 直接访问 RAG 服务 metrics 端点
curl http://localhost:8081/v1/rag/metrics
```

预期输出（Prometheus 格式）：

```
# HELP sentinel_x_rag_queries_total Total number of RAG queries.
# TYPE sentinel_x_rag_queries_total counter
sentinel_x_rag_queries_total 123

# HELP sentinel_x_rag_cache_hit_rate Cache hit rate (0-1).
# TYPE sentinel_x_rag_cache_hit_rate gauge
sentinel_x_rag_cache_hit_rate 0.7500

# HELP sentinel_x_rag_llm_calls_total Total number of LLM calls.
# TYPE sentinel_x_rag_llm_calls_total counter
sentinel_x_rag_llm_calls_total 45
...
```

### 3. 访问 Grafana 仪表盘

1. 访问 http://localhost:3000
2. 使用默认凭证登录（admin/admin）
3. 导航到 "RAG Service" 文件夹
4. 打开 "Sentinel-X RAG Service Monitoring" 仪表盘

## 指标说明

### 查询指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `sentinel_x_rag_queries_total` | Counter | 总查询次数 |
| `sentinel_x_rag_queries_cache_hits_total` | Counter | 缓存命中次数 |
| `sentinel_x_rag_queries_cache_misses_total` | Counter | 缓存未命中次数 |
| `sentinel_x_rag_queries_errors_total` | Counter | 查询错误次数 |
| `sentinel_x_rag_cache_hit_rate` | Gauge | 缓存命中率（0-1） |

### 检索指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `sentinel_x_rag_retrieval_total` | Counter | 总检索次数 |
| `sentinel_x_rag_retrieval_duration_seconds_total` | Counter | 检索总耗时（秒） |
| `sentinel_x_rag_retrieval_errors_total` | Counter | 检索错误次数 |

### LLM 调用指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `sentinel_x_rag_llm_calls_total` | Counter | LLM 总调用次数 |
| `sentinel_x_rag_llm_calls_duration_seconds_total` | Counter | LLM 调用总耗时（秒） |
| `sentinel_x_rag_llm_calls_errors_total` | Counter | LLM 调用错误次数 |
| `sentinel_x_rag_llm_calls_retries_total` | Counter | LLM 重试次数 |
| `sentinel_x_rag_llm_tokens_prompt_total` | Counter | Prompt tokens 总数 |
| `sentinel_x_rag_llm_tokens_completion_total` | Counter | Completion tokens 总数 |

### 熔断器指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `sentinel_x_rag_circuit_breaker_opens_total` | Counter | 熔断器打开次数 |
| `sentinel_x_rag_circuit_breaker_state` | Gauge | 熔断器状态（0=closed, 1=open, 2=half-open） |

### 索引指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `sentinel_x_rag_documents_indexed_total` | Counter | 已索引文档数 |
| `sentinel_x_rag_chunks_indexed_total` | Counter | 已索引分块数 |
| `sentinel_x_rag_index_errors_total` | Counter | 索引错误次数 |

### 运行时指标

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `sentinel_x_rag_uptime_seconds` | Gauge | 服务运行时间（秒） |

## 常用 PromQL 查询

### 查询 QPS

```promql
# 5 分钟平均 QPS
rate(sentinel_x_rag_queries_total[5m])

# 按小时统计查询量
increase(sentinel_x_rag_queries_total[1h])
```

### 错误率

```promql
# 查询错误率
(
  rate(sentinel_x_rag_queries_errors_total[5m])
  /
  rate(sentinel_x_rag_queries_total[5m])
) * 100

# LLM 调用错误率
(
  rate(sentinel_x_rag_llm_calls_errors_total[5m])
  /
  rate(sentinel_x_rag_llm_calls_total[5m])
) * 100
```

### 平均延迟

```promql
# 检索平均延迟
(
  rate(sentinel_x_rag_retrieval_duration_seconds_total[5m])
  /
  rate(sentinel_x_rag_retrieval_total[5m])
)

# LLM 调用平均延迟
(
  rate(sentinel_x_rag_llm_calls_duration_seconds_total[5m])
  /
  rate(sentinel_x_rag_llm_calls_total[5m])
)
```

### 缓存命中率

```promql
# 当前缓存命中率
sentinel_x_rag_cache_hit_rate

# 5 分钟缓存命中率趋势
avg_over_time(sentinel_x_rag_cache_hit_rate[5m])
```

### Token 使用量

```promql
# 每小时 Token 使用量
increase(sentinel_x_rag_llm_tokens_prompt_total[1h])
+ increase(sentinel_x_rag_llm_tokens_completion_total[1h])

# Token 使用速率
rate(sentinel_x_rag_llm_tokens_prompt_total[5m])
+ rate(sentinel_x_rag_llm_tokens_completion_total[5m])
```

## 生产环境最佳实践

### 1. 数据保留策略

```yaml
# Prometheus 配置
global:
  storage:
    tsdb:
      retention.time: 30d     # 保留 30 天
      retention.size: 50GB    # 或 50GB
```

### 2. 高可用配置

- **Prometheus**: 部署多个 Prometheus 实例，使用相同配置
- **Alertmanager**: 集群模式部署，避免单点故障
- **Grafana**: 使用外部数据库（MySQL/PostgreSQL）存储配置

### 3. 安全加固

```yaml
# Prometheus 启用 HTTPS 和认证
global:
  external_labels:
    cluster: 'production'

web:
  tls_config:
    cert_file: /etc/prometheus/tls/server.crt
    key_file: /etc/prometheus/tls/server.key
  basic_auth_users:
    prometheus: $2y$10$...  # bcrypt 哈希密码
```

### 4. 性能优化

- **采集间隔**: 根据业务需求调整（推荐 15-30 秒）
- **抓取超时**: 设置合理超时（推荐 10 秒）
- **并发抓取**: 控制并发数避免过载

```yaml
scrape_configs:
  - job_name: 'rag-service'
    scrape_interval: 15s
    scrape_timeout: 10s
    # 并发抓取控制
    relabel_configs:
      - source_labels: [__address__]
        regex: '.*'
        target_label: __param_collect[]
        replacement: node
```

## 故障排查

### Prometheus 无法抓取指标

**诊断**:
```bash
# 检查 targets 状态
curl http://localhost:9090/api/v1/targets | jq

# 检查 RAG 服务 metrics 端点
curl http://localhost:8081/v1/rag/metrics
```

**常见问题**:
- RAG 服务未启动
- 防火墙阻止连接
- 配置中的地址或端口错误
- metrics 端点返回 404

### Grafana 无数据

**诊断**:
1. 检查 Prometheus 数据源配置
2. 在 Grafana Explore 中手动查询指标
3. 检查 Prometheus 是否成功抓取数据

**常见问题**:
- Prometheus 数据源 URL 错误
- 时间范围选择不当
- PromQL 查询语句错误

### 告警未触发

**诊断**:
```bash
# 检查告警规则状态
curl http://localhost:9090/api/v1/rules | jq

# 检查 Alertmanager 状态
curl http://localhost:9093/api/v1/status | jq
```

**常见问题**:
- 告警规则语法错误
- 阈值设置不合理
- Alertmanager 配置错误
- Slack/邮件配置错误

## 监控清单

部署监控系统前，请确认以下清单：

- [ ] Prometheus 成功启动并抓取指标
- [ ] Grafana 成功连接 Prometheus 数据源
- [ ] 仪表盘显示正常数据
- [ ] 告警规则已加载
- [ ] Alertmanager 成功连接 Slack/邮件
- [ ] 测试告警发送成功
- [ ] 文档已更新（Runbook链接）
- [ ] 团队成员已培训

## 相关文档

- [Grafana 仪表盘配置](./grafana-dashboard.json)
- [告警规则配置](./alerting-rules.md)
- [RAG 服务 API 文档](../api/rag-api.md)
- [Prometheus 官方文档](https://prometheus.io/docs/)
- [Grafana 官方文档](https://grafana.com/docs/)
- [Alertmanager 官方文档](https://prometheus.io/docs/alerting/latest/alertmanager/)

## 总结

完整的监控体系包括：

1. **指标收集**: RAG 服务导出 Prometheus 格式指标
2. **指标存储**: Prometheus 定期抓取并存储
3. **可视化**: Grafana 仪表盘展示关键指标
4. **告警**: Alertmanager 根据规则发送告警
5. **响应**: 团队根据 Runbook 处理告警

通过本指南，您应该能够搭建完整的 RAG 服务监控体系，及时发现和解决问题，确保服务稳定运行。
