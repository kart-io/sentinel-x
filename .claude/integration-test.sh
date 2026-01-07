#!/bin/bash

# Sentinel-X 适配器移除后集成测试脚本

set -e

BASE_URL="http://localhost:8081"
TEST_USER="testuser_$(date +%s)"
TEST_PASSWORD="Test@123456"
TEST_EMAIL="test_$(date +%s)@example.com"

echo "========================================="
echo "Sentinel-X 集成测试开始"
echo "========================================="

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试计数器
PASSED=0
FAILED=0

# 测试函数
test_api() {
    local name=$1
    local method=$2
    local endpoint=$3
    local data=$4
    local expected_code=$5
    local auth_header=$6

    echo -e "\n${YELLOW}测试: $name${NC}"

    if [ -n "$auth_header" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $auth_header" \
            -d "$data")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)

    if [ "$http_code" = "$expected_code" ]; then
        echo -e "${GREEN}✓ PASSED${NC} (HTTP $http_code)"
        echo "响应: $body"
        PASSED=$((PASSED + 1))
        echo "$body"
    else
        echo -e "${RED}✗ FAILED${NC} (期望 HTTP $expected_code, 实际 HTTP $http_code)"
        echo "响应: $body"
        FAILED=$((FAILED + 1))
        echo ""
    fi
}

echo -e "\n========================================="
echo "1. 健康检查和基础端点测试"
echo "========================================="

# 1.1 Health Check
test_api "健康检查" "GET" "/health" "" "200"

# 1.2 Health Readiness
test_api "就绪检查" "GET" "/health/ready" "" "200"

# 1.3 Version Endpoint
test_api "版本端点" "GET" "/version" "" "200"

echo -e "\n========================================="
echo "2. 用户注册流程测试"
echo "========================================="

# 2.1 用户注册
register_response=$(test_api "用户注册" "POST" "/v1/users" \
    "{\"username\":\"$TEST_USER\",\"password\":\"$TEST_PASSWORD\",\"email\":\"$TEST_EMAIL\"}" \
    "200")

echo -e "\n========================================="
echo "3. 用户认证流程测试"
echo "========================================="

# 3.1 用户登录
login_response=$(test_api "用户登录" "POST" "/auth/login" \
    "{\"username\":\"$TEST_USER\",\"password\":\"$TEST_PASSWORD\"}" \
    "200")

# 提取 token
TOKEN=$(echo "$login_response" | jq -r '.data.token // .token // empty')

if [ -z "$TOKEN" ]; then
    echo -e "${RED}✗ 无法获取 Token，跳过后续需要认证的测试${NC}"
else
    echo -e "${GREEN}✓ Token 获取成功${NC}"

    # 3.2 获取当前用户信息
    test_api "获取当前用户信息" "GET" "/auth/me" "" "200" "$TOKEN"

    echo -e "\n========================================="
    echo "4. 用户管理测试（需认证）"
    echo "========================================="

    # 4.1 获取用户列表
    test_api "获取用户列表" "GET" "/v1/users?page=1&page_size=10" "" "200" "$TOKEN"

    # 4.2 获取用户详情
    test_api "获取用户详情" "GET" "/v1/users/detail?username=$TEST_USER" "" "200" "$TOKEN"

    # 4.3 更新用户信息
    test_api "更新用户信息" "PUT" "/v1/users" \
        "{\"username\":\"$TEST_USER\",\"email\":\"updated_$TEST_EMAIL\"}" \
        "200" "$TOKEN"

    echo -e "\n========================================="
    echo "5. 中间件功能测试"
    echo "========================================="

    # 5.1 RequestID 中间件
    echo -e "\n${YELLOW}测试: RequestID 中间件${NC}"
    request_id=$(curl -s -v "$BASE_URL/health" 2>&1 | grep -i "X-Request-ID" || echo "")
    if [ -n "$request_id" ]; then
        echo -e "${GREEN}✓ PASSED${NC} - RequestID 头存在"
        echo "$request_id"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗ FAILED${NC} - RequestID 头不存在"
        FAILED=$((FAILED + 1))
    fi

    # 5.2 CORS 中间件（如果启用）
    echo -e "\n${YELLOW}测试: CORS 中间件${NC}"
    cors_response=$(curl -s -X OPTIONS "$BASE_URL/v1/users" \
        -H "Origin: http://example.com" \
        -H "Access-Control-Request-Method: POST" \
        -v 2>&1 | grep -i "access-control-allow" || echo "")
    if [ -n "$cors_response" ]; then
        echo -e "${GREEN}✓ PASSED${NC} - CORS 头存在"
        echo "$cors_response"
        PASSED=$((PASSED + 1))
    else
        echo -e "${YELLOW}⚠ SKIPPED${NC} - CORS 可能未启用"
    fi

    echo -e "\n========================================="
    echo "6. 错误处理测试"
    echo "========================================="

    # 6.1 404 错误
    test_api "404 错误处理" "GET" "/nonexistent" "" "404"

    # 6.2 无效 Token
    test_api "无效 Token" "GET" "/auth/me" "" "401" "invalid_token"

    # 6.3 参数验证错误
    test_api "参数验证错误" "POST" "/v1/users" \
        "{\"username\":\"\",\"password\":\"\"}" \
        "400"

    echo -e "\n========================================="
    echo "7. 清理测试数据"
    echo "========================================="

    # 7.1 删除测试用户
    test_api "删除测试用户" "DELETE" "/v1/users?username=$TEST_USER" "" "200" "$TOKEN"

    # 7.2 登出
    test_api "用户登出" "POST" "/auth/logout" "" "200" "$TOKEN"
fi

echo -e "\n========================================="
echo "测试总结"
echo "========================================="
echo -e "${GREEN}通过: $PASSED${NC}"
echo -e "${RED}失败: $FAILED${NC}"
echo "总计: $((PASSED + FAILED))"

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}所有测试通过！✓${NC}"
    exit 0
else
    echo -e "\n${RED}存在失败的测试！✗${NC}"
    exit 1
fi
