#!/bin/bash
set -e

BASE_URL="http://localhost:8081"
USERNAME="testuser_$(date +%s)"
PASSWORD="password123"
EMAIL="${USERNAME}@example.com"
ROLE_CODE="testrole_$(date +%s)"

echo "=== 1. Register User ${USERNAME} ==="
curl -s -X POST "${BASE_URL}/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"password\": \"${PASSWORD}\", \"email\": \"${EMAIL}\"}" | jq .

echo -e "\n=== 2. Login & Get Token ==="
LOGIN_RESP=$(curl -s -X POST "${BASE_URL}/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"password\": \"${PASSWORD}\"}")
echo "${LOGIN_RESP}" | jq .

TOKEN=$(echo "${LOGIN_RESP}" | jq -r '.data.token')
if [ "${TOKEN}" == "null" ]; then
  echo "Failed to get token"
  exit 1
fi
echo "Token: ${TOKEN:0:20}..."

AUTH_HEADER="Authorization: Bearer ${TOKEN}"

echo -e "\n=== 3. Get Profile (Protected) ==="
curl -s -X GET "${BASE_URL}/auth/me" -H "${AUTH_HEADER}" | jq .

echo -e "\n=== 4. List Users ==="
curl -s -X GET "${BASE_URL}/v1/users" -H "${AUTH_HEADER}" | jq .

echo -e "\n=== 5. Get User Details ==="
curl -s -X GET "${BASE_URL}/v1/users/detail?username=${USERNAME}" -H "${AUTH_HEADER}" | jq .

echo -e "\n=== 6. Update User (Body Parameters) ==="
curl -s -X PUT "${BASE_URL}/v1/users" \
  -H "${AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"email\": \"updated_${EMAIL}\", \"mobile\": \"13800138000\"}" | jq .

echo -e "\n=== 7. Create Role ${ROLE_CODE} ==="
curl -s -X POST "${BASE_URL}/v1/roles" \
  -H "${AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d "{\"code\": \"${ROLE_CODE}\", \"name\": \"Test Role\"}" | jq .

echo -e "\n=== 8. List Roles ==="
curl -s -X GET "${BASE_URL}/v1/roles" -H "${AUTH_HEADER}" | jq .

echo -e "\n=== 9. Assign Role to User ==="
curl -s -X POST "${BASE_URL}/v1/users/roles?username=${USERNAME}" \
  -H "${AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"role_code\": \"${ROLE_CODE}\"}" | jq .

echo -e "\n=== 10. Get User Roles ==="
curl -s -X GET "${BASE_URL}/v1/users/roles?username=${USERNAME}" -H "${AUTH_HEADER}" | jq .

echo -e "\n=== 11. Update Password (Secure Method - All in Body) ==="
curl -s -X POST "${BASE_URL}/v1/users/password" \
  -H "${AUTH_HEADER}" \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"${USERNAME}\", \"old_password\": \"${PASSWORD}\", \"new_password\": \"NewPassword123\", \"confirm_password\": \"NewPassword123\"}" | jq .

echo -e "\n=== Verification Complete ==="
