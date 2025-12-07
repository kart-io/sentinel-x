#!/bin/bash

# Gin + Dynamic Config Reload 测试脚本

echo "=== Gin + Dynamic Config Reload 测试 ==="

# 检查服务器是否运行
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "❌ 请先启动服务器: go run main.go"
    exit 1
fi

echo "✅ 服务器已运行"

# 1. 获取当前配置
echo
echo "1. 获取当前配置"
curl -s http://localhost:8080/config/current | jq '.current_config'

# 2. 测试 API 接口
echo
echo "2. 测试基本 API 接口"
curl -s http://localhost:8080/ | jq '.message'
curl -s http://localhost:8080/users/123 | jq '.name'

# 3. 通过 API 重载配置 (切换到 Zap + DEBUG)
echo
echo "3. 通过 API 重载配置 (切换到 Zap + DEBUG)"
curl -s -X POST http://localhost:8080/config/reload \
  -H "Content-Type: application/json" \
  -d '{
    "engine": "zap",
    "level": "DEBUG",
    "format": "console",
    "output_paths": ["stdout"],
    "development": true
  }' | jq '.message'

sleep 1

# 4. 验证配置已更改
echo
echo "4. 验证配置已更改"
curl -s http://localhost:8080/config/current | jq '.current_config.engine, .current_config.level'

# 5. 测试调试日志 (现在应该可见)
echo
echo "5. 测试调试日志 (应该可见 DEBUG 级别)"
curl -s http://localhost:8080/health > /dev/null

# 6. 回滚到之前的配置
echo
echo "6. 回滚到之前的配置"
curl -s -X POST http://localhost:8080/config/rollback | jq '.message'

sleep 1

# 7. 验证回滚
echo
echo "7. 验证回滚结果"
curl -s http://localhost:8080/config/current | jq '.current_config.engine, .current_config.level'

# 8. 查看备份配置
echo
echo "8. 查看备份配置"
curl -s http://localhost:8080/config/backups | jq '.backup_count'

# 9. 测试错误处理
echo
echo "9. 测试错误和恢复"
curl -s http://localhost:8080/error | jq '.error'
curl -s http://localhost:8080/panic | jq '.error'

# 10. 测试慢请求日志
echo
echo "10. 测试慢请求日志 (3秒延迟)"
curl -s http://localhost:8080/slow | jq '.message'

echo
echo "✅ 所有测试完成！"
echo
echo "💡 手动测试:"
echo "   - 编辑 logger-config.yaml 文件查看自动重载"
echo "   - 发送 SIGUSR1 信号: kill -USR1 $(pgrep -f gin-reload)"
echo "   - 通过 API 尝试不同的配置组合"