# 配置文件迁移验证清单

## 1. 文件完整性检查

- [x] configs/user-center.yaml - 已转换为扁平化格式
- [x] configs/user-center-dev.yaml - 已转换为扁平化格式
- [x] configs/sentinel-api.yaml - 已转换为扁平化格式
- [x] configs/sentinel-api-dev.yaml - 已转换为扁平化格式
- [x] configs/rag.yaml - 已添加 version 配置段
- [x] configs/auth.yaml - 无需修改（已是扁平化结构）

## 2. YAML 语法验证

- [x] configs/user-center.yaml - ✓ 语法正确
- [x] configs/user-center-dev.yaml - ✓ 语法正确
- [x] configs/sentinel-api.yaml - ✓ 语法正确
- [x] configs/sentinel-api-dev.yaml - ✓ 语法正确
- [x] configs/rag.yaml - ✓ 语法正确
- [x] configs/auth.yaml - ✓ 语法正确

## 3. 结构一致性验证

### 所有文件都包含以下顶层配置段

- [x] `server:` - 服务器基础配置
- [x] `http:` - HTTP 服务器配置
- [x] `grpc:` - gRPC 服务器配置（如适用）
- [x] `metrics:` - Metrics 中间件配置
- [x] `health:` - Health Check 中间件配置
- [x] `pprof:` - Pprof 中间件配置
- [x] `recovery:` - Recovery 中间件配置
- [x] `logger:` - Logger 中间件配置
- [x] `cors:` - CORS 中间件配置
- [x] `timeout:` - Timeout 中间件配置
- [x] `request-id:` - Request ID 中间件配置
- [x] `version:` - Version 端点配置（新增）
- [x] `log:` - 日志配置
- [x] 业务特定配置（`jwt:`, `mysql:`, `redis:` 等）

### 移除的嵌套结构

- [x] 已移除 `server.http.middleware` 嵌套结构
- [x] 已移除 `middleware.enabled` 列表
- [x] 已移除 `middleware.disable-*` 标志

## 4. 配置值保留验证

### user-center.yaml

- [x] HTTP 端口：:8081
- [x] gRPC 端口：:8101
- [x] Server Mode：both
- [x] Metrics Subsystem：user-center
- [x] JWT disable-auth：true
- [x] MySQL password：""（空，生产环境）
- [x] Redis password：""（空，生产环境）
- [x] Version enabled：true ✅

### user-center-dev.yaml

- [x] HTTP 端口：:8081
- [x] gRPC 端口：:8104
- [x] Server Mode：both
- [x] Metrics Subsystem：user-center
- [x] JWT disable-auth：true
- [x] MySQL password："root123"
- [x] Redis password："redis_pass"
- [x] Version enabled：true ✅

### sentinel-api.yaml

- [x] HTTP 端口：:8080
- [x] gRPC 端口：:8100
- [x] Server Mode：both
- [x] Metrics Subsystem：api
- [x] JWT disable-auth：false
- [x] Auth 配置：包含 token-lookup, skip-paths 等
- [x] MySQL password："root123"
- [x] Redis password："redis_pass"
- [x] Version enabled：true ✅

### sentinel-api-dev.yaml

- [x] HTTP 端口：:8100
- [x] gRPC 端口：:8103
- [x] Server Mode：http
- [x] Metrics Subsystem：api
- [x] JWT disable-auth：false
- [x] MySQL host：""（禁用）
- [x] Redis host：""（禁用）
- [x] Version enabled：true ✅

### rag.yaml

- [x] HTTP 端口：:8082
- [x] gRPC 端口：:8102
- [x] Server Mode：http
- [x] Metrics Subsystem：rag
- [x] Milvus 配置：完整保留
- [x] Embedding 配置：完整保留
- [x] Chat 配置：完整保留
- [x] RAG 配置：完整保留
- [x] Cache 配置：完整保留
- [x] Version enabled：true ✅

## 5. 中间件配置完整性验证

### 所有文件都包含以下中间件配置

- [x] Metrics：path, namespace, subsystem
- [x] Health：path, liveness-path, readiness-path
- [x] Pprof：prefix, enable-cmdline, enable-profile, enable-symbol, enable-trace
- [x] Recovery：enable-stack-trace
- [x] Logger：skip-paths, use-structured-logger
- [x] CORS：allow-origins, allow-methods, allow-headers, allow-credentials, max-age
- [x] Timeout：timeout, skip-paths
- [x] Request ID：header
- [x] Version：enabled, path, hide-details

### Sentinel API 特殊配置

- [x] Auth：token-lookup, auth-scheme, skip-paths, skip-path-prefixes

## 6. 文档完整性验证

- [x] .claude/config-migration-report.md - 详细迁移报告
- [x] .claude/config-migration-comparison.md - 迁移对比详情
- [x] .claude/config-migration-summary.md - 完成总结
- [x] .claude/config-quick-reference.md - 快速参考
- [x] .claude/verification-checklist.md - 验证清单（本文件）

## 7. 代码适配待办事项

### 优先级 1 - 必须立即完成

- [ ] 更新配置结构体定义
  - 文件：`internal/bootstrap/config.go` 或类似文件
  - 任务：移除 `MiddlewareConfig`，添加顶层中间件配置字段
  - 影响：配置加载失败

- [ ] 更新配置加载逻辑
  - 文件：`internal/bootstrap/config.go`
  - 任务：移除 `server.http.middleware` 访问路径
  - 影响：编译错误

- [ ] 更新中间件注册代码
  - 文件：`internal/bootstrap/middleware.go` 或类似文件
  - 任务：移除 `enabled` 列表和 `disable-*` 标志检查
  - 影响：中间件可能无法正常工作

### 优先级 2 - 尽快完成

- [ ] 实现 version 端点
  - 文件：`internal/*/router/router.go`
  - 任务：添加 `/version` 端点处理逻辑
  - 影响：新端点无法访问

- [ ] 添加配置验证逻辑
  - 文件：`internal/bootstrap/config.go`
  - 任务：验证必要的中间件配置存在
  - 影响：配置错误可能导致运行时问题

- [ ] 更新单元测试
  - 文件：`internal/bootstrap/*_test.go`
  - 任务：更新配置相关测试用例
  - 影响：测试失败

### 优先级 3 - 后续优化

- [ ] 更新集成测试
  - 文件：`test/integration/*`
  - 任务：使用新配置格式
  - 影响：集成测试可能失败

- [ ] 更新文档
  - 文件：`README.md`, `docs/`
  - 任务：更新配置说明
  - 影响：文档过时

- [ ] 添加配置迁移工具
  - 文件：新建 `tools/migrate-config.go`
  - 任务：自动转换旧配置格式
  - 影响：无（可选工具）

## 8. 测试验证待办事项

### 配置加载测试

- [ ] 测试 user-center 配置加载
  ```bash
  go run cmd/user-center/main.go --config configs/user-center.yaml --dry-run
  ```

- [ ] 测试 user-center-dev 配置加载
  ```bash
  go run cmd/user-center/main.go --config configs/user-center-dev.yaml --dry-run
  ```

- [ ] 测试 sentinel-api 配置加载
  ```bash
  go run cmd/api/main.go --config configs/sentinel-api.yaml --dry-run
  ```

- [ ] 测试 sentinel-api-dev 配置加载
  ```bash
  go run cmd/api/main.go --config configs/sentinel-api-dev.yaml --dry-run
  ```

### 服务启动测试

- [ ] 启动 user-center（生产配置）
  ```bash
  go run cmd/user-center/main.go --config configs/user-center.yaml
  ```

- [ ] 启动 user-center（开发配置）
  ```bash
  go run cmd/user-center/main.go --config configs/user-center-dev.yaml
  ```

- [ ] 启动 sentinel-api（生产配置）
  ```bash
  go run cmd/api/main.go --config configs/sentinel-api.yaml
  ```

- [ ] 启动 sentinel-api（开发配置）
  ```bash
  go run cmd/api/main.go --config configs/sentinel-api-dev.yaml
  ```

### 端点功能测试

#### User Center (:8081)

- [ ] 测试 Health Check
  ```bash
  curl http://localhost:8081/health
  curl http://localhost:8081/live
  curl http://localhost:8081/ready
  ```

- [ ] 测试 Metrics
  ```bash
  curl http://localhost:8081/metrics
  ```

- [ ] 测试 Version（新增）
  ```bash
  curl http://localhost:8081/version
  ```

#### Sentinel API (:8080 或 :8100)

- [ ] 测试 Health Check
  ```bash
  curl http://localhost:8080/health
  curl http://localhost:8080/live
  curl http://localhost:8080/ready
  ```

- [ ] 测试 Metrics
  ```bash
  curl http://localhost:8080/metrics
  ```

- [ ] 测试 Version（新增）
  ```bash
  curl http://localhost:8080/version
  ```

### 中间件功能测试

- [ ] 测试 CORS
  ```bash
  curl -i -H "Origin: http://example.com" \
       -H "Access-Control-Request-Method: POST" \
       -X OPTIONS http://localhost:8081/api/v1/users
  ```

- [ ] 测试 Request ID
  ```bash
  curl -i http://localhost:8081/api/v1/users | grep X-Request-ID
  ```

- [ ] 测试超时
  ```bash
  time curl http://localhost:8081/slow-endpoint
  ```

- [ ] 测试 Recovery（触发 panic）
  ```bash
  curl http://localhost:8081/panic-endpoint
  ```

- [ ] 测试日志
  ```bash
  tail -f logs/*.log
  # 发起请求后检查日志输出
  curl http://localhost:8081/api/v1/users
  ```

### 单元测试

- [ ] 运行所有单元测试
  ```bash
  go test ./... -v
  ```

- [ ] 运行配置相关测试
  ```bash
  go test ./internal/bootstrap/... -v
  ```

### 集成测试

- [ ] 运行集成测试
  ```bash
  go test ./test/integration/... -v
  ```

## 9. 性能验证

- [ ] 配置加载性能
  ```bash
  time go run cmd/user-center/main.go --config configs/user-center.yaml --dry-run
  ```

- [ ] 服务启动时间
  ```bash
  time go run cmd/user-center/main.go --config configs/user-center.yaml
  ```

- [ ] 端点响应时间
  ```bash
  curl -w "@curl-format.txt" http://localhost:8081/health
  ```

## 10. 安全验证

- [ ] 确认生产配置不包含硬编码密码
  ```bash
  grep -r "password:" configs/*.yaml | grep -v '""'
  ```

- [ ] 确认生产配置不包含硬编码密钥
  ```bash
  grep -r "key:" configs/*.yaml | grep -v '""'
  ```

- [ ] 确认 CORS 配置适合生产环境
  ```bash
  grep -A3 "cors:" configs/user-center.yaml configs/sentinel-api.yaml
  ```

## 11. 文档验证

- [ ] 所有文档文件存在
- [ ] 所有文档文件格式正确（Markdown）
- [ ] 所有文档文件内容完整
- [ ] 所有代码示例可运行

## 12. Git 提交验证

- [ ] 所有配置文件已暂存
  ```bash
  git status configs/
  ```

- [ ] 所有文档文件已暂存
  ```bash
  git status .claude/
  ```

- [ ] 提交信息准备完整
- [ ] 提交信息包含 Co-Authored-By

## 13. 回滚准备

- [ ] 备份原始配置文件
  ```bash
  cp configs/*.yaml configs/*.yaml.backup
  ```

- [ ] 记录当前 Git commit
  ```bash
  git rev-parse HEAD > .claude/last-commit.txt
  ```

- [ ] 准备回滚脚本
  ```bash
  # 见 .claude/config-migration-comparison.md
  ```

## 验证完成标准

### 必须项（全部完成才能发布）

- [x] 所有配置文件 YAML 语法正确
- [x] 所有配置值保留不变
- [x] 所有文档文件生成完整
- [ ] 配置加载代码适配完成
- [ ] 中间件注册代码适配完成
- [ ] 所有单元测试通过
- [ ] 服务能够正常启动
- [ ] 所有端点能够正常访问

### 可选项（建议完成）

- [ ] 集成测试通过
- [ ] 性能验证通过
- [ ] 文档更新完成
- [ ] 配置迁移工具开发完成

## 签名确认

### 配置文件迁移

- [x] 配置文件转换完成：Claude Sonnet 4.5
- [x] 配置文件验证完成：Claude Sonnet 4.5
- [x] 文档生成完成：Claude Sonnet 4.5

### 代码适配

- [ ] 配置结构体更新完成：_______
- [ ] 配置加载逻辑更新完成：_______
- [ ] 中间件注册代码更新完成：_______

### 测试验证

- [ ] 单元测试验证完成：_______
- [ ] 集成测试验证完成：_______
- [ ] 端点功能验证完成：_______

### 最终审核

- [ ] 代码审查完成：_______
- [ ] 技术负责人批准：_______
- [ ] 可以合并到主分支：_______

---

**清单创建时间**：2026-01-06
**清单创建者**：Claude Sonnet 4.5
**配置格式版本**：扁平化格式 v1.0
