#!/bin/bash

# 创建 OTLP 网络
# Create OTLP Docker network for service communication

set -e

echo "🌐 创建 Docker 网络..."

# 检查网络是否已存在
if docker network inspect kart-otlp-network >/dev/null 2>&1; then
    echo "✅ 网络 kart-otlp-network 已存在"
else
    # 创建网络
    docker network create kart-otlp-network
    echo "✅ 网络 kart-otlp-network 创建成功"
fi

echo "🌐 网络配置完成"