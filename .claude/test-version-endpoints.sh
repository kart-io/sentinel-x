#!/bin/bash

# 版本端点测试脚本
# 用于验证所有三个服务的版本端点是否正常工作

set -e

echo "========================================"
echo "版本端点测试脚本"
echo "========================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查服务是否运行
check_service() {
    local service_name=$1
    local port=$2

    if nc -z localhost $port 2>/dev/null; then
        echo -e "${GREEN}✓${NC} $service_name 服务运行在端口 $port"
        return 0
    else
        echo -e "${RED}✗${NC} $service_name 服务未运行在端口 $port"
        return 1
    fi
}

# 测试版本端点
test_version_endpoint() {
    local service_name=$1
    local port=$2
    local url="http://localhost:$port/version"

    echo ""
    echo "----------------------------------------"
    echo "测试 $service_name 版本端点: $url"
    echo "----------------------------------------"

    if ! check_service "$service_name" "$port"; then
        echo -e "${YELLOW}跳过测试${NC}"
        return 1
    fi

    # 发送请求
    response=$(curl -s -w "\n%{http_code}" "$url")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    echo "HTTP 状态码: $http_code"

    if [ "$http_code" == "200" ]; then
        echo -e "${GREEN}✓ 版本端点响应成功${NC}"
        echo ""
        echo "响应内容:"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        return 0
    else
        echo -e "${RED}✗ 版本端点响应失败${NC}"
        echo "响应内容: $body"
        return 1
    fi
}

# 主测试流程
main() {
    local failed=0

    echo "开始测试所有服务的版本端点..."
    echo ""

    # 测试 user-center (端口 8081)
    if ! test_version_endpoint "user-center" 8081; then
        ((failed++))
    fi

    # 测试 api (端口 8080)
    if ! test_version_endpoint "api" 8080; then
        ((failed++))
    fi

    # 测试 rag (端口 8082)
    if ! test_version_endpoint "rag" 8082; then
        ((failed++))
    fi

    # 输出总结
    echo ""
    echo "========================================"
    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}✓ 所有测试通过${NC}"
    else
        echo -e "${RED}✗ $failed 个测试失败${NC}"
    fi
    echo "========================================"

    return $failed
}

# 运行主函数
main
exit $?
