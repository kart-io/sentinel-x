# Sentinel-X Makefile 指南

本文档提供了 Sentinel-X 项目中可用 `Makefile` 命令的综合指南。该 `Makefile` 旨在简化常见的开发、构建、测试和部署任务。

## 用法

不带参数运行 `make` 将默认执行 `all` 目标（构建所有二进制文件）。

```bash
make <TARGET> <OPTIONS>
```

要查看带有描述的所有可用目标列表，请运行：

```bash
make help
```

要查看按定义文件分组的详细目标列表，请运行：

```bash
make targets
```

## 选项

这些选项可以附加到 `make` 命令后以自定义行为。

| 选项 | 描述 | 默认值 | 示例 |
| :--- | :--- | :--- | :--- |
| `BINS` | 要构建的二进制文件。 | `cmd/` 下的所有文件 | `make build BINS="api"` |
| `IMAGES` | 要构建的后端镜像。 | `cmd/` 下的所有文件 | `make image IMAGES="api"` |
| `V` | 详细模式。设置为 1 以启用。 | 0 | `make build V=1` |

## 命令

### 构建 (Build)

| 目标 | 描述 |
| :--- | :--- |
| `all` | 构建所有二进制文件。 |
| `build` | 为宿主平台构建源代码。依赖于 `tidy`。 |
| `clean` | 清理构建产物（`_output/` 目录）。 |
| `deps` | 安装依赖项和开发工具。 |
| `tidy` | 运行 `go mod tidy` 和 `go mod vendor`。 |
| `update` | 使用 `hack/update-vendor.sh` 更新 vendor 依赖。 |

### 测试与质量 (Test & Quality)

| 目标 | 描述 |
| :--- | :--- |
| `test` | 运行单元测试。 |
| `test-coverage` | 运行单元测试并生成覆盖率报告。 |
| `lint` | 运行 `golangci-lint` 检查代码规范错误。依赖于 `tidy`。 |
| `fmt` | 使用 `gofumpt`、`goimports` 和 `gci` 格式化源代码。 |
| `verify-sonic` | 验证 Sonic JSON 集成状态。 |

### 镜像 (Image)

| 目标 | 描述 |
| :--- | :--- |
| `image` | 构建 Docker 镜像。 |
| `image.multiarch` | 构建多架构 Docker 镜像 (amd64, arm64)。 |
| `push` | 构建 Docker 镜像并推送到镜像仓库。 |
| `push.multiarch` | 构建/推送多架构镜像到镜像仓库。 |

### 代码生成 (Code Generation)

| 目标 | 描述 |
| :--- | :--- |
| `gen.proto` | 从 Protocol Buffers 生成 Go 代码 (`pkg/api`)。 |
| `gen.clean` | 删除生成的 protobuf 文件（`.pb.go`、`.swagger.json` 等）。 |
| `clean.proto` | `gen.clean` 的别名。 |

### 数据管理 (Data Management)

| 目标 | 描述 |
| :--- | :--- |
| `data.download.milvus` | 下载 Milvus 文档压缩包。 |
| `data.extract.milvus` | 解压 Milvus 文档到项目目录。 |
| `data.setup.milvus` | 一键下载并解压 Milvus 文档（推荐）。 |
| `data.clean.milvus` | 清理已下载的 Milvus 文档和压缩包。 |

### 开发 (Development)

| 目标 | 描述 |
| :--- | :--- |
| `run` | 运行服务器。默认为 `api`。用法：`make run BIN=user-center`。 |
| `run-api` | 使用开发配置在本地运行 API 服务器。 |
| `run-user-center`| 使用开发配置在本地运行 User Center 服务器。 |
| `run.go` | 直接使用 `go run` 运行服务（无需编译）。用法：`make run.go BIN=user-center`。 |
| `deploy.infra` | 启动基础依赖服务（MySQL, Redis）。用法：`make deploy.infra`。 |
| `deploy.run` | 使用 Docker Compose 启动所有服务。 |
| `deploy.down` | 停止并移除所有 Docker Compose 服务。 |
| `run-example` | 运行示例服务器。 |
| `update-goagent` | 从上游同步 `goagent` 代码。 |
| `update-logger` | 从上游同步 `logger` 代码。 |

### 辅助 (Helper)

| 目标 | 描述 |
| :--- | :--- |
| `help` | 显示分类的帮助信息。 |
| `targets` | 显示按源文件分组的所有目标。 |

## 详细工作流

### 环境设置

开始时，运行 `deps` 安装必要的工具，如 `golangci-lint`、`mockgen` 和 `protoc` 插件。

```bash
make deps
```

### 代码开发

1.  **修改代码**：进行更改。
2.  **格式化**：运行 `make fmt` 以确保代码格式正确并整理导入。
    ```bash
    make fmt
    ```
3.  **Lint 检查**：运行 `make lint` 检查常见错误。
    ```bash
    make lint
    ```
4.  **测试**：运行测试以确保没有回归。
    ```bash
    make test
    ```
5.  **运行**：在本地运行服务以验证行为。
    ```bash
    make run-api
    ```

### 使用 Protobuf

如果你修改了 `pkg/api` 中的文件（Protobuf 定义）：

1.  重新生成代码：
    ```bash
    make gen.proto
    ```
2.  如果需要清理旧文件：
    ```bash
    make clean.proto
    ```

### 构建和部署镜像

为 API 服务器构建 Docker 镜像：

```bash
make image IMAGES="api"
```

为 AMD64 和 ARM64 构建并推送镜像：

```bash
make push.multiarch IMAGES="api"
```

### 高级目标 (Advanced Targets)

这些目标在子 Makefile 中可用，通常由主目标使用，或用于特定的细粒度任务。

| 目标 | 描述 | 来源 |
| :--- | :--- | :--- |
| `go.build.<PLATFORM>.<BINARY>` | 为特定平台构建二进制文件。 | `golang.mk` |
| `go.clean` | 清理构建产物。 | `golang.mk` |
| `go.test` | 运行单元测试。 | `golang.mk` |
| `go.test.cover` | 运行带有覆盖率的单元测试。 | `golang.mk` |
| `go.fmt` | 格式化源代码。 | `golang.mk` |
| `go.lint` | 运行 linters。 | `golang.mk` |
| `tools.install` | 安装所有工具。 | `tools.mk` |
| `tools.install.<TOOL>` | 安装特定工具。 | `tools.mk` |
| `tools.verify.<TOOL>` | 验证特定工具是否已安装。 | `tools.mk` |
| `image.verify` | 验证 docker 版本。 | `image.mk` |
| `image.daemon.verify` | 验证 docker 守护进程版本。 | `image.mk` |
| `image.dockerfile` | 生成所有 dockerfiles。 | `image.mk` |
| `image.dockerfile.<IMAGE>` | 为特定镜像生成 Dockerfile。 | `image.mk` |
| `image.build.<PLATFORM>.<IMAGE>` | 构建指定的 docker 镜像。 | `image.mk` |
| `image.push.<PLATFORM>.<IMAGE>` | 构建并推送指定的 docker 镜像。 | `image.mk` |
| `image.push.<PLATFORM>.<IMAGE>` | 构建并推送指定的 docker 镜像。 | `image.mk` |
| `run` | 运行默认服务器 (api)。支持 `ENV`。 | `run.mk` |
| `run.<BINARY>` | 运行指定服务器。自动检测配置文件。 | `run.mk` |
| `run.go` | 直接运行指定服务器（无需编译）。 | `run.mk` |
| `deploy.run` | 启动所有服务。 | `deploy.mk` |
| `deploy.down` | 停止所有服务。 | `deploy.mk` |
| `deploy.infra` | 仅启动基础服务。 | `deploy.mk` |
| `gen.proto` | 生成 Proto 代码。 | `gen.mk` |
| `gen.clean` | 清理生成的 protobuf 文件。 | `gen.mk` |
| `update` | 更新 vendor 依赖。 | `update.mk` |
| `update-goagent` | 从上游同步 goagent。 | `update.mk` |
| `update-logger` | 从上游同步 logger。 | `update.mk` |
| `data.download.milvus` | 下载 Milvus 文档。 | `data.mk` |
| `data.extract.milvus` | 解压 Milvus 文档。 | `data.mk` |
| `data.setup.milvus` | 下载并解压 Milvus 文档。 | `data.mk` |
| `data.clean.milvus` | 清理 Milvus 文档。 | `data.mk` |

### 辅助脚本 (Helper Scripts)

`scripts/` 目录包含支撑 Makefile 的辅助脚本。虽然通常由 `make` 调用，但它们也可以直接运行。

#### `scripts/image.sh`

镜像操作的包装器。

```bash
# 构建镜像（默认）
./scripts/image.sh build

# 推送镜像
./scripts/image.sh push
```

#### `scripts/buildx.sh`

多架构镜像操作的包装器。

```bash
# 构建多架构镜像
./scripts/buildx.sh build

# 推送多架构镜像
./scripts/buildx.sh push
```

#### `scripts/install/protobuf.sh`

安装 Protocol Buffers 工具（buf, protoc-gen-go 等）。

```bash
./scripts/install/protobuf.sh
```

#### `scripts/gen-dockerfile.sh`

生成项目的 Dockerfiles。由 `make image.dockerfile` 使用。

```bash
./scripts/gen-dockerfile.sh <OUTPUT_DIR> <IMAGE_NAME>
```

#### `scripts/make-rules/run.mk`

运行命令。

#### `scripts/make-rules/update.mk`

更新命令。

#### `scripts/make-rules/tools.mk`

工具安装。

#### `scripts/install/install.sh`

工具安装的核心脚本。由 `make tools.install` 使用。

```bash
./scripts/install/install.sh [TOOL_NAME]
```

#### `scripts/verify-sonic.sh`

验证 Sonic JSON 集成。由 `make verify-sonic` 使用。

```bash
./scripts/verify-sonic.sh
```
