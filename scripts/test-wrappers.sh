#!/usr/bin/env bash

# 扩展测试脚本 - 测试部分更新和清空字段功能

BASE_URL="http://localhost:8080"
USERNAME="testuser_$(date +%s)"
PASSWORD="TestPass123"
EMAIL="${USERNAME}@example.com"
ROLE_CODE="testrole_$(date +%s)"

echo "===  Protobuf Wrappers 测试 ==="

# 1. 注册用户
echo -e "\n=== 1. Register User ===\"
REGISTER_RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"email\": \"${EMAIL}\", \"password\": \"${PASSWORD}\"}")
echo "$REGISTER_RESPONSE" | jq .

# 2. 登录获取 Token
echo -e "\n=== 2. Login ===\"
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"password\": \"${PASSWORD}\"}")
echo "$LOGIN_RESPONSE" | jq .

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.data.token')
AUTH_HEADER="Authorization: Bearer ${TOKEN}"

# 3. 创建用户，设置 email 和 mobile
echo -e "\n=== 3. Get Profile（初始状态）===\"
curl -s -X GET "${BASE_URL}/v1/auth/profile" -H "${AUTH_HEADER}" | jq .

# 4. 测试：只更新 email（部分更新）
echo -e "\n=== 4. Update - 只更新 email（部分更新）===\"
curl -s -X PUT "${BASE_URL}/v1/users" \
  -H "${AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"email\": \"partial_${EMAIL}\"}" | jq .

# 5. 验证 mobile 未被改变
echo -e "\n=== 5. Get Profile（验证 mobile 未变）===\"
curl -s -X GET "${BASE_URL}/v1/auth/profile" -H "${AUTH_HEADER}" | jq .

# 6. 测试：设置 mobile
echo -e "\n=== 6. Update - 设置 mobile ===\"
curl -s -X PUT "${BASE_URL}/v1/users" \
  -H "${AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"mobile\": \"13800138000\"}" | jq .

# 7. 验证 email 保持上一次的值
echo -e "\n=== 7. Get Profile（验证 email 保持）===\"
curl -s -X GET "${BASE_URL}/v1/auth/profile" -H "${AUTH_HEADER}" | jq .

# 8. 测试：清空 email
echo -e "\n=== 8. Update - 清空 email ===\"
curl -s -X PUT "${BASE_URL}/v1/users" \
  -H "${AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"email\": \"\"}" | jq .

# 9. 验证 email 已清空，mobile 保持不变
echo -e "\n=== 9. Get Profile（验证 email 清空，mobile 保持）===\"
curl -s -X GET "${BASE_URL}/v1/auth/profile" -H "${AUTH_HEADER}" | jq .

# 10. 测试：同时更新 email 和 mobile
echo -e "\n=== 10. Update - 同时更新 email 和 mobile ===\"
curl -s -X PUT "${BASE_URL}/v1/users" \
  -H "${AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"email\": \"final_${EMAIL}\", \"mobile\": \"13900000000\"}" | jq .

# 11. 最终验证
echo -e "\n=== 11. Get Profile（最终状态）===\"
curl -s -X GET "${BASE_URL}/v1/auth/profile" -H "${AUTH_HEADER}" | jq .

echo -e "\n=== Protobuf Wrappers 测试完成 ==="
