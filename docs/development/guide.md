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

*   **添加新依赖**:
    ```bash
    go get github.com/some/lib
    go mod vendor
    ```
*   **更新依赖**:
    ```bash
    go get -u github.com/some/lib
    go mod vendor
    ```

请务必提交 `go.mod`、`go.sum` 以及 `vendor/` 目录的变更。
