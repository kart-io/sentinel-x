# 框架适配器移除 - 快速执行指南

## 执行概览

**目标**: 完全移除框架适配器抽象层，直接使用 Gin 框架
**预计时间**: 3-4 个工作日（19-29 小时）
**风险级别**: 中等（有完整的测试和回滚方案）

## 前置准备

### 1. 环境检查

```bash
# 确认 Go 版本
go version  # 需要 >= 1.25.0

# 确认当前分支
git branch --show-current  # 应该在 master

# 确认工作区干净
git status  # 应该没有未提交的更改

# 确认测试通过
make test

# 确认编译成功
make build
make build-user-center
```

### 2. 创建工作分支

```bash
git checkout -b refactor/remove-adapter-abstraction
```

### 3. 运行测试基线

```bash
# 记录当前测试状态
make test > .claude/test-baseline.log 2>&1

# 记录当前编译状态
make build > .claude/build-baseline.log 2>&1
```

### 4. 备份关键文件

```bash
mkdir -p .claude/backup
cp -r pkg/infra/server/transport/http .claude/backup/
cp -r pkg/infra/adapter .claude/backup/
cp -r internal/user-center/handler .claude/backup/
cp -r pkg/infra/middleware .claude/backup/
```

## 阶段性执行清单

### 阶段 0: 理解当前架构（30 分钟）

- [ ] 阅读 `.claude/adapter-removal-analysis.md` 完整分析报告
- [ ] 理解当前调用链路和依赖关系
- [ ] 明确迁移目标和验收标准

### 阶段 1: Response 工具迁移（2-3 小时）

**任务**:

- [ ] 修改 `pkg/utils/response/writer.go`
  - 将所有 `transport.Context` 替换为 `*gin.Context`
  - 更新方法签名
  - 更新实现逻辑

- [ ] 修改 `internal/pkg/httputils/response.go`
  - 更新 `WriteResponse` 函数签名
  - 适配新的 Context 类型

**验证**:

```bash
go test ./pkg/utils/response/... -v
go test ./internal/pkg/httputils/... -v
```

**提交**:

```bash
git add .
git commit -m "refactor(response): 迁移 Response 工具到直接使用 gin.Context"
```

### 阶段 2: 中间件迁移（4-6 小时）

#### P0: 基础中间件（1-2 小时）

**任务**:

- [ ] `pkg/infra/middleware/request_id.go`
  - 签名: `func(...) gin.HandlerFunc`
  - 参数: `c *gin.Context`
  - Next: `c.Next()`

- [ ] `pkg/infra/middleware/resilience/recovery.go`
  - 同上

- [ ] `pkg/infra/middleware/observability/logger.go`
  - 同上

**验证**:

```bash
go test ./pkg/infra/middleware/request_id_test.go -v
go test ./pkg/infra/middleware/resilience/recovery_test.go -v
go test ./pkg/infra/middleware/observability/logger_test.go -v
```

**提交**:

```bash
git add pkg/infra/middleware/request_id.go
git commit -m "refactor(middleware): 迁移 RequestID 中间件到 Gin"

git add pkg/infra/middleware/resilience/recovery.go
git commit -m "refactor(middleware): 迁移 Recovery 中间件到 Gin"

git add pkg/infra/middleware/observability/logger.go
git commit -m "refactor(middleware): 迁移 Logger 中间件到 Gin"
```

#### P1: 安全中间件（1-2 小时）

**任务**:

- [ ] `pkg/infra/middleware/security/cors.go`
- [ ] `pkg/infra/middleware/security/security_headers.go`
- [ ] `pkg/infra/middleware/auth/auth.go`
- [ ] `pkg/infra/middleware/auth/authz.go`

**验证**:

```bash
go test ./pkg/infra/middleware/security/... -v
go test ./pkg/infra/middleware/auth/... -v
```

**提交**:

```bash
git add pkg/infra/middleware/security/
git commit -m "refactor(middleware): 迁移安全中间件到 Gin"

git add pkg/infra/middleware/auth/
git commit -m "refactor(middleware): 迁移认证中间件到 Gin"
```

#### P2: 功能中间件（1-2 小时）

**任务**:

- [ ] `pkg/infra/middleware/resilience/timeout.go`
- [ ] `pkg/infra/middleware/resilience/ratelimit.go`
- [ ] `pkg/infra/middleware/resilience/circuit_breaker.go`
- [ ] `pkg/infra/middleware/resilience/body_limit.go`
- [ ] `pkg/infra/middleware/performance/compression.go`
- [ ] `pkg/infra/middleware/observability/metrics.go`
- [ ] `pkg/infra/middleware/observability/tracing.go`

**验证**:

```bash
go test ./pkg/infra/middleware/resilience/... -v
go test ./pkg/infra/middleware/performance/... -v
go test ./pkg/infra/middleware/observability/... -v
```

**提交**:

```bash
git add pkg/infra/middleware/
git commit -m "refactor(middleware): 迁移所有功能中间件到 Gin"
```

#### P3: 辅助中间件（30 分钟）

**任务**:

- [ ] `pkg/infra/middleware/version.go`
- [ ] `pkg/infra/middleware/health.go`
- [ ] `pkg/infra/middleware/pprof.go`

**验证**:

```bash
go test ./pkg/infra/middleware/... -v
```

**提交**:

```bash
git add pkg/infra/middleware/
git commit -m "refactor(middleware): 迁移辅助中间件到 Gin"
```

### 阶段 3: Handler 层迁移（3-4 小时）

#### 3.1 Auth Handler（1 小时）

**任务**:

- [ ] 修改 `internal/user-center/handler/auth.go`
  - 所有方法签名: `func (h *AuthHandler) XXX(c *gin.Context)`
  - 替换: `c.ShouldBindAndValidate` → `c.ShouldBindJSON` + `validator.Global().Validate`
  - 替换: `c.Request()` → `c.Request.Context()`
  - 替换: `httputils.WriteResponse` → 更新参数类型

**验证**:

```bash
go test ./internal/user-center/handler/auth_test.go -v
```

**提交**:

```bash
git add internal/user-center/handler/auth.go
git commit -m "refactor(handler): 迁移 AuthHandler 到 Gin"
```

#### 3.2 Role Handler（1 小时）

**任务**:

- [ ] 修改 `internal/user-center/handler/role.go`（同上）

**验证**:

```bash
go test ./internal/user-center/handler/role_test.go -v
```

**提交**:

```bash
git add internal/user-center/handler/role.go
git commit -m "refactor(handler): 迁移 RoleHandler 到 Gin"
```

#### 3.3 User Handler（1-2 小时）

**任务**:

- [ ] 修改 `internal/user-center/handler/user.go`（同上，17 个方法）

**验证**:

```bash
go test ./internal/user-center/handler/user_test.go -v
```

**提交**:

```bash
git add internal/user-center/handler/user.go
git commit -m "refactor(handler): 迁移 UserHandler 到 Gin"
```

### 阶段 4: Router 层迁移（1-2 小时）

**任务**:

- [ ] 修改 `internal/user-center/router/router.go`
  - `router.Handle(method, path, handler)` → `group.POST/GET/PUT/DELETE(path, handler)`
  - `router.Use(middleware...)` → `group.Use(middleware...)`
  - 更新所有路由注册代码

**验证**:

```bash
go test ./internal/user-center/router/... -v
make build
```

**提交**:

```bash
git add internal/user-center/router/router.go
git commit -m "refactor(router): 迁移 Router 到直接使用 Gin"
```

### 阶段 5: Server 核心重构（2-3 小时）

**任务**:

- [ ] 修改 `pkg/infra/server/transport/http/server.go`
  - 替换字段: `adapter Adapter` → `engine *gin.Engine`
  - 简化 `NewServer` 构造函数
  - 添加 `Engine() *gin.Engine` 方法
  - 移除 `Router()` 方法（或改为返回 `*gin.RouterGroup`）
  - 简化 `applyMiddleware` 方法

- [ ] 修改 `pkg/infra/server/transport/transport.go`
  - 移除 `Context` 接口
  - 移除 `HandlerFunc` 类型
  - 移除 `MiddlewareFunc` 类型
  - 保留 `Transport`, `HTTPRegistrar`, `HTTPHandler` 接口

- [ ] 修改 `pkg/options/server/http/options.go`
  - 移除 `AdapterType` 定义
  - 移除 `Adapter` 字段

**验证**:

```bash
go test ./pkg/infra/server/... -v
make build
make build-user-center
```

**提交**:

```bash
git add pkg/infra/server/transport/http/server.go
git add pkg/infra/server/transport/transport.go
git add pkg/options/server/http/options.go
git commit -m "refactor(server): 移除 Adapter 抽象，直接使用 Gin Engine"
```

### 阶段 6: 清理和优化（2-3 小时）

**任务**:

- [ ] 删除废弃文件

```bash
git rm -r pkg/infra/adapter/
git rm pkg/infra/server/transport/http/adapter.go
git rm pkg/infra/server/transport/http/adapter_test.go
git rm pkg/infra/server/transport/http/bridge.go
```

- [ ] 移除适配器导入
  - 编辑 `internal/user-center/server.go`
  - 移除: `_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"`
  - 移除: `_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/echo"`

- [ ] 更新配置示例
  - 编辑 `configs/user-center.yaml`
  - 移除 `http.adapter` 字段

**验证**:

```bash
make build
make test
```

**提交**:

```bash
git add .
git commit -m "refactor(cleanup): 删除适配器抽象层相关代码"
```

### 阶段 7: 完整测试（2-3 小时）

**任务**:

- [ ] 运行完整测试套件

```bash
make test
```

- [ ] 测试覆盖率

```bash
make test-coverage
```

- [ ] 启动服务并测试

```bash
make run-dev
```

- [ ] 集成测试

```bash
bash .claude/integration-test.sh
```

- [ ] 性能测试

```bash
wrk -t4 -c100 -d30s http://localhost:8081/health
```

- [ ] 生成迁移报告

```bash
bash .claude/generate-migration-report.sh
```

### 阶段 8: 文档更新（1-2 小时）

**任务**:

- [ ] 更新 `docs/design/architecture.md`
- [ ] 更新 `docs/design/user-center.md`
- [ ] 更新 `CLAUDE.md`
- [ ] 创建 `CHANGELOG.md` 条目

**提交**:

```bash
git add docs/ CLAUDE.md CHANGELOG.md
git commit -m "docs: 更新架构文档，反映适配器移除后的新架构"
```

### 阶段 9: 合并和部署（1 小时）

**任务**:

- [ ] 推送分支

```bash
git push origin refactor/remove-adapter-abstraction
```

- [ ] 创建 Pull Request

```bash
gh pr create --title "重构：移除框架适配器抽象层" \
  --body "$(cat .claude/pr-description.md)"
```

- [ ] Code Review
- [ ] 合并到 master

```bash
git checkout master
git merge refactor/remove-adapter-abstraction
```

- [ ] 打标签

```bash
git tag -a v1.1.0 -m "移除框架适配器抽象层，简化架构"
git push --tags
```

## 常见问题处理

### 问题 1: 编译错误 - 类型不匹配

**现象**:

```
cannot use xxx (type func(transport.Context)) as type gin.HandlerFunc
```

**解决**:

检查函数签名是否正确更新为 `func(c *gin.Context)`

### 问题 2: 测试失败 - Context 方法不存在

**现象**:

```
c.ShouldBindAndValidate undefined
```

**解决**:

替换为 `c.ShouldBindJSON(&req)` 并手动调用 `validator.Global().Validate(&req)`

### 问题 3: 中间件执行顺序错误

**现象**:

中间件没有按预期顺序执行

**解决**:

检查中间件注册顺序，确保在 `NewServer` 时正确应用

### 问题 4: Response 格式变化

**现象**:

API 响应格式与之前不一致

**解决**:

检查 `response.Success` 和 `response.Fail` 的实现，确保格式一致

## 回滚方案

### 快速回滚

```bash
# 回到 master
git checkout master

# 删除工作分支
git branch -D refactor/remove-adapter-abstraction

# 重新构建
make build
```

### 部分回滚

```bash
# 恢复特定文件
cp .claude/backup/pkg/infra/server/transport/http/* \
   pkg/infra/server/transport/http/

# 重新提交
git add .
git commit -m "revert: 恢复部分文件"
```

## 成功标准

### 必须满足（Blocking）

- [x] 所有单元测试通过
- [x] 所有集成测试通过
- [x] 编译无错误无警告
- [x] API 行为保持一致
- [x] 错误响应格式一致
- [x] 性能无退化（或有提升）

### 建议满足（Non-blocking）

- [ ] 测试覆盖率 ≥ 80%
- [ ] 性能提升 ≥ 5%
- [ ] 代码行数减少 ≥ 1000 行
- [ ] 文档完整更新

## 联系方式

如果在执行过程中遇到问题：

1. 查看 `.claude/adapter-removal-analysis.md` 详细分析
2. 检查 `.claude/operations-log.md` 操作记录
3. 运行 `.claude/generate-migration-report.sh` 生成诊断报告

---

**文档版本**: 1.0
**最后更新**: 2026-01-07
**适用版本**: Sentinel-X v1.x
