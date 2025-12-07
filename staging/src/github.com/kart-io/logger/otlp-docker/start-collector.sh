#!/bin/bash

# 启动 OTEL Collector 数据处理服务
# Start OTEL Collector data processing service

set -e

echo "🔧 启动 OTEL Collector..."

# 检查配置文件是否存在
if [ ! -f "$(pwd)/otel-collector-config.yaml" ]; then
    echo "❌ 配置文件 otel-collector-config.yaml 不存在"
    exit 1
fi

# 停止并删除现有容器
if docker ps -a --format "table {{.Names}}" | grep -q "^kart-otel-collector$"; then
    echo "🔄 停止现有 OTEL Collector 容器..."
    docker stop kart-otel-collector >/dev/null 2>&1 || true
    docker rm kart-otel-collector >/dev/null 2>&1 || true
fi

# 启动 OTEL Collector 容器
docker run -d \
    --name kart-otel-collector \
    --network kart-otlp-network \
    --hostname otel-collector \
    -p 4317:4317 \
    -p 4318:4318 \
    -p 13134:13133 \
    -p 1778:1777 \
    -p 8889:8888 \
    -v "$(pwd)/otel-collector-config.yaml:/etc/otelcol-contrib/otel-collector-config.yaml:ro" \
    -e GOMEMLIMIT=512MiB \
    otel/opentelemetry-collector-contrib:0.132.0 \
    --config=/etc/otelcol-contrib/otel-collector-config.yaml

echo "⏳ 等待 OTEL Collector 启动..."
sleep 5

# 健康检查
if curl -s http://localhost:13134/ >/dev/null; then
    echo "✅ OTEL Collector 启动成功"
    echo "   - gRPC: localhost:4317"
    echo "   - HTTP: localhost:4318" 
    echo "   - 健康检查: http://localhost:13134"
    echo "   - 指标: http://localhost:8889/metrics"
else
    echo "❌ OTEL Collector 健康检查失败"
    exit 1
fi