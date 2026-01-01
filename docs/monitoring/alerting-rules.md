# Sentinel-X RAG 服务监控告警规则

本文档定义 RAG 服务的 Prometheus 告警规则，用于及时发现和响应系统异常。

## 告警规则配置

### 1. 查询相关告警

#### 高错误率告警
```yaml
groups:
  - name: rag_query_alerts
    interval: 30s
    rules:
      # 查询错误率过高
      - alert: RAGQueryHighErrorRate
        expr: |
          (
            rate(sentinel_x_rag_queries_errors_total[5m])
            /
            rate(sentinel_x_rag_queries_total[5m])
          ) > 0.05
        for: 2m
        labels:
          severity: critical
          component: rag
          service: query
        annotations:
          summary: "RAG 查询错误率过高"
          description: "过去 5 分钟查询错误率为 {{ $value | humanizePercentage }}，超过 5% 阈值"
          runbook: "https://docs.sentinel-x.io/runbooks/rag-high-error-rate"

      # 查询 QPS 突增
      - alert: RAGQueryHighQPS
        expr: rate(sentinel_x_rag_queries_total[5m]) > 100
        for: 5m
        labels:
          severity: warning
          component: rag
          service: query
        annotations:
          summary: "RAG 查询 QPS 异常高"
          description: "当前 QPS 为 {{ $value | humanize }}，超过 100 QPS 阈值，可能存在异常流量"
          runbook: "https://docs.sentinel-x.io/runbooks/rag-high-qps"

      # 查询 QPS 异常低（可能服务故障）
      - alert: RAGQueryLowQPS
        expr: rate(sentinel_x_rag_queries_total[5m]) < 0.1
        for: 10m
        labels:
          severity: warning
          component: rag
          service: query
        annotations:
          summary: "RAG 查询 QPS 异常低"
          description: "当前 QPS 为 {{ $value | humanize }}，低于 0.1 QPS，可能服务已停止响应"
          runbook: "https://docs.sentinel-x.io/runbooks/rag-low-qps"
```

### 2. 缓存相关告警

```yaml
groups:
  - name: rag_cache_alerts
    interval: 30s
    rules:
      # 缓存命中率过低
      - alert: RAGCacheLowHitRate
        expr: sentinel_x_rag_cache_hit_rate < 0.3
        for: 10m
        labels:
          severity: warning
          component: rag
          service: cache
        annotations:
          summary: "RAG 缓存命中率过低"
          description: "当前缓存命中率为 {{ $value | humanizePercentage }}，低于 30%，缓存效果不佳"
          runbook: "https://docs.sentinel-x.io/runbooks/rag-low-cache-hit-rate"
```

### 3. LLM 调用相关告警

```yaml
groups:
  - name: rag_llm_alerts
    interval: 30s
    rules:
      # LLM 调用错误率过高
      - alert: RAGLLMHighErrorRate
        expr: |
          (
            rate(sentinel_x_rag_llm_calls_errors_total[5m])
            /
            rate(sentinel_x_rag_llm_calls_total[5m])
          ) > 0.1
        for: 2m
        labels:
          severity: critical
          component: rag
          service: llm
        annotations:
          summary: "LLM 调用错误率过高"
          description: "过去 5 分钟 LLM 调用错误率为 {{ $value | humanizePercentage }}，超过 10% 阈值"
          runbook: "https://docs.sentinel-x.io/runbooks/llm-high-error-rate"

      # LLM 调用平均耗时过长
      - alert: RAGLLMHighLatency
        expr: |
          (
            rate(sentinel_x_rag_llm_calls_duration_seconds_total[5m])
            /
            rate(sentinel_x_rag_llm_calls_total[5m])
          ) > 30
        for: 5m
        labels:
          severity: warning
          component: rag
          service: llm
        annotations:
          summary: "LLM 调用平均耗时过长"
          description: "过去 5 分钟 LLM 调用平均耗时为 {{ $value | humanizeDuration }}，超过 30 秒"
          runbook: "https://docs.sentinel-x.io/runbooks/llm-high-latency"

      # LLM 重试次数过多
      - alert: RAGLLMHighRetries
        expr: rate(sentinel_x_rag_llm_calls_retries_total[5m]) > 1
        for: 5m
        labels:
          severity: warning
          component: rag
          service: llm
        annotations:
          summary: "LLM 调用重试次数过多"
          description: "过去 5 分钟 LLM 调用重试率为 {{ $value | humanize }}/s，可能存在网络或服务问题"
          runbook: "https://docs.sentinel-x.io/runbooks/llm-high-retries"

      # LLM Token 使用量突增（成本告警）
      - alert: RAGLLMHighTokenUsage
        expr: |
          (
            rate(sentinel_x_rag_llm_tokens_prompt_total[5m])
            +
            rate(sentinel_x_rag_llm_tokens_completion_total[5m])
          ) > 100000
        for: 10m
        labels:
          severity: warning
          component: rag
          service: llm
          type: cost
        annotations:
          summary: "LLM Token 使用量异常高"
          description: "过去 5 分钟 Token 使用率为 {{ $value | humanize }}/s，可能导致成本激增"
          runbook: "https://docs.sentinel-x.io/runbooks/llm-high-token-usage"
```

### 4. 熔断器相关告警

```yaml
groups:
  - name: rag_circuit_breaker_alerts
    interval: 30s
    rules:
      # 熔断器打开
      - alert: RAGCircuitBreakerOpen
        expr: sentinel_x_rag_circuit_breaker_state == 1
        for: 1m
        labels:
          severity: critical
          component: rag
          service: circuit_breaker
        annotations:
          summary: "RAG 熔断器已打开"
          description: "熔断器已打开，LLM 调用已被阻断，服务处于降级状态"
          runbook: "https://docs.sentinel-x.io/runbooks/circuit-breaker-open"

      # 熔断器频繁打开
      - alert: RAGCircuitBreakerFlapping
        expr: changes(sentinel_x_rag_circuit_breaker_state[10m]) > 5
        for: 2m
        labels:
          severity: warning
          component: rag
          service: circuit_breaker
        annotations:
          summary: "RAG 熔断器状态不稳定"
          description: "过去 10 分钟熔断器状态变化 {{ $value }} 次，系统不稳定"
          runbook: "https://docs.sentinel-x.io/runbooks/circuit-breaker-flapping"
```

### 5. 检索相关告警

```yaml
groups:
  - name: rag_retrieval_alerts
    interval: 30s
    rules:
      # 检索错误率过高
      - alert: RAGRetrievalHighErrorRate
        expr: |
          (
            rate(sentinel_x_rag_retrieval_errors_total[5m])
            /
            rate(sentinel_x_rag_retrieval_total[5m])
          ) > 0.05
        for: 2m
        labels:
          severity: critical
          component: rag
          service: retrieval
        annotations:
          summary: "向量检索错误率过高"
          description: "过去 5 分钟检索错误率为 {{ $value | humanizePercentage }}，超过 5% 阈值"
          runbook: "https://docs.sentinel-x.io/runbooks/retrieval-high-error-rate"

      # 检索平均耗时过长
      - alert: RAGRetrievalHighLatency
        expr: |
          (
            rate(sentinel_x_rag_retrieval_duration_seconds_total[5m])
            /
            rate(sentinel_x_rag_retrieval_total[5m])
          ) > 5
        for: 5m
        labels:
          severity: warning
          component: rag
          service: retrieval
        annotations:
          summary: "向量检索平均耗时过长"
          description: "过去 5 分钟检索平均耗时为 {{ $value | humanizeDuration }}，超过 5 秒"
          runbook: "https://docs.sentinel-x.io/runbooks/retrieval-high-latency"
```

### 6. 索引相关告警

```yaml
groups:
  - name: rag_index_alerts
    interval: 30s
    rules:
      # 索引错误率过高
      - alert: RAGIndexHighErrorRate
        expr: rate(sentinel_x_rag_index_errors_total[5m]) > 0.5
        for: 2m
        labels:
          severity: warning
          component: rag
          service: index
        annotations:
          summary: "文档索引错误率过高"
          description: "过去 5 分钟索引错误率为 {{ $value | humanize }}/s"
          runbook: "https://docs.sentinel-x.io/runbooks/index-high-error-rate"
```

## 告警级别说明

### Critical（严重）
- **影响**: 服务不可用或严重降级
- **响应时间**: 立即响应（<5分钟）
- **通知渠道**: PagerDuty + Slack + 短信 + 电话
- **示例**:
  - 查询错误率 > 5%
  - LLM 调用错误率 > 10%
  - 熔断器打开

### Warning（警告）
- **影响**: 性能下降或资源使用异常
- **响应时间**: 工作时间内响应（<30分钟）
- **通知渠道**: Slack + 邮件
- **示例**:
  - 查询 QPS 异常
  - 缓存命中率过低
  - LLM 耗时过长

### Info（信息）
- **影响**: 需要关注但不影响服务
- **响应时间**: 工作时间内查看
- **通知渠道**: Slack
- **示例**:
  - 服务重启
  - 配置变更

## 告警抑制规则

```yaml
# 告警抑制配置（避免告警风暴）
inhibit_rules:
  # 熔断器打开时，抑制 LLM 错误告警
  - source_match:
      alertname: RAGCircuitBreakerOpen
    target_match:
      alertname: RAGLLMHighErrorRate
    equal: ['component', 'service']

  # 查询错误率高时，抑制检索和 LLM 错误告警
  - source_match:
      alertname: RAGQueryHighErrorRate
    target_match_re:
      alertname: (RAGRetrievalHighErrorRate|RAGLLMHighErrorRate)
    equal: ['component']
```

## 告警路由规则

```yaml
# Alertmanager 路由配置
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
      receiver: 'pagerduty-critical'
      continue: true

    # Warning 告警路由
    - match:
        severity: warning
      receiver: 'slack-warnings'

    # 成本告警特殊路由
    - match:
        type: cost
      receiver: 'slack-cost-alerts'

receivers:
  - name: 'default'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#rag-monitoring'

  - name: 'pagerduty-critical'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'

  - name: 'slack-warnings'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#rag-warnings'

  - name: 'slack-cost-alerts'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#cost-alerts'
```

## 告警响应 Runbook

### RAG 高错误率

**诊断步骤**:
1. 检查 `/v1/rag/metrics` 端点，查看具体错误指标
2. 检查 `/v1/rag/stats` 端点，查看服务整体状态
3. 查看应用日志，定位具体错误原因

**常见原因**:
- Milvus 向量数据库不可用
- LLM API 服务故障或限流
- Redis 缓存连接失败
- 网络问题

**修复措施**:
1. 检查依赖服务健康状态
2. 检查 API key 是否过期或被限流
3. 检查网络连接
4. 如果是暂时性问题，重启服务

### LLM 调用高延迟

**诊断步骤**:
1. 检查 LLM 服务商状态页面
2. 查看 Grafana "LLM 调用平均耗时" 面板
3. 检查网络延迟

**常见原因**:
- LLM 服务负载高
- 网络延迟增加
- prompt 过长
- 并发请求过多

**修复措施**:
1. 优化 prompt 长度
2. 增加超时控制
3. 启用重试机制（已默认启用）
4. 考虑增加熔断阈值
5. 联系 LLM 服务商

### 熔断器打开

**诊断步骤**:
1. 检查 LLM 调用错误日志
2. 查看熔断器统计信息
3. 检查 LLM 服务可用性

**修复措施**:
1. 解决 LLM 服务问题
2. 等待熔断器自动恢复（默认 60 秒）
3. 如需手动恢复，重启服务

## 指标阈值调整

根据实际业务场景，可能需要调整以下阈值：

| 指标 | 默认阈值 | 调整建议 |
|------|----------|----------|
| 查询错误率 | 5% | 根据 SLA 调整 |
| 缓存命中率 | 30% | 根据业务特性调整 |
| LLM 错误率 | 10% | 根据 LLM 服务稳定性调整 |
| LLM 平均耗时 | 30s | 根据用户体验要求调整 |
| 检索平均耗时 | 5s | 根据向量数据库性能调整 |

## 参考资料

- [Prometheus 告警规则文档](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
- [Alertmanager 配置文档](https://prometheus.io/docs/alerting/latest/configuration/)
- [Grafana 告警文档](https://grafana.com/docs/grafana/latest/alerting/)
