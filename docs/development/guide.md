# 开发指南

本项目采用了类似于 Kubernetes 的 Monorepo（单体仓库）架构。本文档旨在说明如何进行代码开发、依赖管理以及核心 `goagent` 模块的贡献流程。

## 项目结构

*   **`staging/src/github.com/kart-io/goagent`**: 这是 `goagent` 库的 **代码真相源 (Source of Truth)**。它是一个普通的目录，而不是子模块 (submodule)。所有的开发和修改都应直接在此目录下进行。
*   **`staging/src/github.com/kart-io/logger`**: 这是 `logger` 库的 **代码真相源 (Source of Truth)**。它是一个普通的目录，而不是子模块 (submodule)。所有的开发和修改都应直接在此目录下进行。
*   **`vendor/`**: 包含项目所有的依赖（包括第三方库）。该目录**已提交到 Git**，以确保构建环境的一致性 (Hermetic Builds)。
*   **`go.mod`**: 使用 `replace` 指令将 `github.com/kart-io/goagent` 和 `github.com/kart-io/logger` 指向本地的 `./staging/src/github.com/kart-io/goagent` 和 `./staging/src/github.com/kart-io/logger` 目录。

## 开发工作流

### 1. 修改核心库 (`goagent`, `logger`)

由于 `goagent` 和 `logger` 是 Monorepo 的一部分，您可以直接在 `staging/src/github.com/kart-io/goagent` 或 `staging/src/github.com/kart-io/logger` 目录下修改代码。

*   **无需 `go get`**: 由于本地 `replace` 指令的存在，您的修改会**立即**在 `sentinel-x` 项目中生效。
*   **原子提交**: 您可以在一次 Git Commit 中同时提交 `sentinel-x` (业务逻辑) 和 `goagent` (库逻辑) 的修改。

### 2. 运行测试

*   **运行所有测试 (sentinel-x)**:
    ```bash
    go test ./...
    ```
    *注意: 这主要运行主模块包的测试。*

*   **运行 `goagent` 或 `logger` 测试**:
    由于 `goagent` 和 `logger` 在技术上是独立的模块（被本地替换），您必须进入其目录运行测试：
    ```bash
    cd staging/src/github.com/kart-io/goagent && go test ./...
    # 或者
    cd staging/src/github.com/kart-io/logger && go test ./...
    ```

### 3. 同步上游核心库 (`goagent`, `logger`) 代码

如果独立的 `goagent` 或 `logger` 仓库有其他人提交了更新，您可以通过以下命令拉取变更到本地 Monorepo。

**警告**: 此操作会覆盖 `staging/` 目录下所有**未提交**的本地修改。

*   **同步 `goagent`**:
    ```bash
    make update-goagent
    ```
*   **同步 `logger`**:
    ```bash
    make update-logger
    ```

这些命令会执行脚本来：
1. 克隆远程仓库到临时目录。
2. 用远程内容替换本地 `staging/` 目录的内容。
3. 运行 `go mod tidy` 和 `go mod vendor`。

### 4. 发布变更到上游核心库 (`goagent`, `logger`)

如果您在本地修改了 `goagent` 或 `logger` 并希望将这些变更推送到独立的仓库（例如供社区其他用户使用），请使用发布脚本。

**前提条件**: 您必须拥有对应远程仓库的 SSH 推送权限 (例如 `git@github.com:kart-io/goagent.git` 或 `git@github.com:kart-io/logger.git`)。

*   **发布 `goagent`**:
    ```bash
    make publish-goagent
    ```
*   **发布 `logger`**:
    ```bash
    make publish-logger
    ```

这些命令会执行脚本来：
1. 将远程仓库克隆到临时目录。
2. 将本地 `staging/` 的内容复制过去。
3. 创建同步 Commit。
4. 推送到远程仓库的 `master` 分支。

## 依赖管理

我们将所有依赖项都纳入版本控制。

*   **更新依赖并同步 vendor**:
    ```bash
    make tidy
    ```
    该命令执行 `go mod tidy && go mod vendor`，整理依赖并更新 vendor 目录。

*   **添加新依赖**:
    ```bash
    go get github.com/some/lib
    make tidy
    ```
*   **更新依赖**:
    ```bash
    go get -u github.com/some/lib
    make tidy
    ```

请务必提交 `go.mod`、`go.sum` 以及 `vendor/` 目录的变更。

## 运行示例服务器

项目提供了一个完整的示例服务器，展示了 Sentinel-X 框架的核心功能。

### 启动示例服务器

```bash
make run-example
```

或者直接运行：

```bash
go run example/server/example/main.go -c example/server/example/configs/sentinel-example.yaml
```

### 服务器配置

示例服务器启动后将监听以下端口：

| 服务 | 端口 | 说明 |
|------|------|------|
| HTTP | 8081 | 使用 Gin 适配器 |
| gRPC | 9091 | 支持 Reflection |

### 已启用的中间件

- **Recovery**: 异常恢复
- **RequestID**: 请求 ID 生成
- **Logger**: 结构化日志
- **Health**: 健康检查
- **Metrics**: Prometheus 指标

### 可用端点

#### API 端点

```bash
# GET 请求
curl http://localhost:8081/api/v1/hello?name=World

# POST 请求
curl -X POST http://localhost:8081/api/v1/hello
```

#### 健康检查端点

```bash
# 健康检查
curl http://localhost:8081/health

# 存活探针
curl http://localhost:8081/live

# 就绪探针
curl http://localhost:8081/ready
```

#### 监控端点

```bash
# Prometheus 指标
curl http://localhost:8081/metrics
```

#### gRPC 端点

```bash
# 使用 grpcurl 调用
grpcurl -plaintext localhost:9091 api.hello.v1.HelloService/SayHello
```

### 命令行参数

示例服务器支持丰富的命令行参数，可通过 `-h` 查看：

```bash
go run example/server/example/main.go -h
```

常用参数：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-c, --config` | - | 配置文件路径 |
| `--http.addr` | :8080 | HTTP 监听地址 |
| `--http.adapter` | gin | HTTP 框架 (gin/echo) |
| `--grpc.addr` | :9090 | gRPC 监听地址 |
| `--server.mode` | both | 服务模式 (http/grpc/both) |
| `--log.level` | INFO | 日志级别 |
| `--log.engine` | slog | 日志引擎 (zap/slog) |
| `--log.format` | json | 日志格式 (json/console) |
