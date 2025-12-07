#!/bin/bash

# 启动 VictoriaLogs 日志存储服务
# Start VictoriaLogs log storage service

set -e

echo "📊 启动 VictoriaLogs..."

# 停止并删除现有容器
if docker ps -a --format "table {{.Names}}" | grep -q "^kart-victorialogs$"; then
    echo "🔄 停止现有 VictoriaLogs 容器..."
    docker stop kart-victorialogs >/dev/null 2>&1 || true
    docker rm kart-victorialogs >/dev/null 2>&1 || true
fi

# 启动 VictoriaLogs 容器
docker run -d \
    --name kart-victorialogs \
    --network kart-otlp-network \
    --hostname victorialogs \
    -p 9428:9428 \
    -e VM_loggerLevel=INFO \
    victoriametrics/victoria-logs:v1.28.0 \
    --storageDataPath=/victoria-logs-data \
    --httpListenAddr=:9428 \
    --retentionPeriod=30d

echo "⏳ 等待 VictoriaLogs 启动..."
sleep 5

# 健康检查
if curl -s http://localhost:9428/health >/dev/null; then
    echo "✅ VictoriaLogs 启动成功 - http://localhost:9428"
else
    echo "❌ VictoriaLogs 健康检查失败"
    exit 1
fi