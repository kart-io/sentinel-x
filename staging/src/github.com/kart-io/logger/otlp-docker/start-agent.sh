#!/bin/bash

# 启动 OTEL Agent 轻量级代理服务
# Start OTEL Agent lightweight proxy service

set -e

echo "🚀 启动 OTEL Agent..."

# 检查配置文件是否存在
if [ ! -f "$(pwd)/otel-agent-config.yaml" ]; then
    echo "❌ 配置文件 otel-agent-config.yaml 不存在"
    exit 1
fi

# 停止并删除现有容器
if docker ps -a --format "table {{.Names}}" | grep -q "^kart-otel-agent$"; then
    echo "🔄 停止现有 OTEL Agent 容器..."
    docker stop kart-otel-agent >/dev/null 2>&1 || true
    docker rm kart-otel-agent >/dev/null 2>&1 || true
fi

# 启动 OTEL Agent 容器
docker run -d \
    --name kart-otel-agent \
    --network kart-otlp-network \
    --hostname otel-agent \
    -p 4327:4317 \
    -p 4328:4318 \
    -p 13133:13133 \
    -p 1777:1777 \
    -p 8888:8888 \
    -v "$(pwd)/otel-agent-config.yaml:/etc/otelcol-contrib/otel-agent-config.yaml:ro" \
    -e GOMEMLIMIT=256MiB \
    otel/opentelemetry-collector-contrib:0.132.0 \
    --config=/etc/otelcol-contrib/otel-agent-config.yaml

echo "⏳ 等待 OTEL Agent 启动..."
sleep 5

# 健康检查
if curl -s http://localhost:13133/ >/dev/null; then
    echo "✅ OTEL Agent 启动成功"
    echo "   - gRPC: localhost:4327" 
    echo "   - HTTP: localhost:4328"
    echo "   - 健康检查: http://localhost:13133"
    echo "   - 指标: http://localhost:8888/metrics"
else
    echo "❌ OTEL Agent 健康检查失败"
    exit 1
fi