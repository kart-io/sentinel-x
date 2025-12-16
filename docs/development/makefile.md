# Sentinel-X Makefile Guide

This document provides a comprehensive guide to the `Makefile` commands available in the Sentinel-X project. The `Makefile` is designed to simplify common development, building, testing, and deployment tasks.

## Usage

Running `make` without arguments will default to the `all` target (which builds all binaries).

```bash
make <TARGET> <OPTIONS>
```

To see a list of all available targets with descriptions, run:

```bash
make help
```

To see a detailed list of targets grouped by the file they are defined in, run:

```bash
make targets
```

## Options

These options can be appended to `make` commands to customize behavior.

| Option | Description | Default | Example |
| :--- | :--- | :--- | :--- |
| `BINS` | The binaries to build. | All in `cmd/` | `make build BINS="api"` |
| `IMAGES` | Backend images to build. | All in `cmd/` | `make image IMAGES="api"` |
| `V` | Verbose mode. Set to 1 to enable. | 0 | `make build V=1` |

## Commands

### Build

| Target | Description |
| :--- | :--- |
| `all` | Build all binaries. |
| `build` | Build source code for host platform. Depends on `tidy`. |
| `clean` | Clean build artifacts (`_output/` directory). |
| `deps` | Install dependencies and development tools. |
| `tidy` | Run `go mod tidy` and `go mod vendor`. |
| `update` | Update vendor dependencies using `hack/update-vendor.sh`. |

### Test & Quality

| Target | Description |
| :--- | :--- |
| `test` | Run unit tests. |
| `test-coverage` | Run unit tests and generate coverage report. |
| `lint` | Run `golangci-lint` to check for linting errors. Depends on `tidy`. |
| `fmt` | Format source code using `gofumpt`, `goimports`, and `gci`. |
| `verify-sonic` | Verify Sonic JSON integration. |

### Image

| Target | Description |
| :--- | :--- |
| `image` | Build docker images. |
| `image.multiarch` | Build docker images for multiple platforms (amd64, arm64). |
| `push` | Build docker images and push to registry. |
| `push.multiarch` | Build/push multi-arch images to registry. |

### Code Generation

| Target | Description |
| :--- | :--- |
| `gen.proto` | Generate Protobuf code. This command automatically checks and installs necessary tools (`buf`, `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-validate`, `protoc-gen-openapiv2`, `protoc-go-inject-tag`). Additionally, it automatically runs `protoc-go-inject-tag` to inject tags into generated Go structs based on `@gotags` comments in `.proto` files. |
| `gen.k8s` | Generate Kubernetes code (clientset, listers, informers, deepcopy). Requires `GROUPS_VERSIONS` argument. Optional `GENERATORS` to filter types (e.g. `GENERATORS="deepcopy,client"`). |
| `gen.clean` | Remove generated protobuf files (`.pb.go`, `.swagger.json`, etc.). |
| `clean.proto` | Alias for `gen.clean`. |

### Development

| Target | Description |
| :--- | :--- |
| `run` | Run a server. Default is `api`. Usage: `make run BIN=user-center`. |
| `run-api` | Run the API server locally with dev config. |
| `run-user-center`| Run the User Center server locally with dev config. |
| `run.go` | Run a server directly using `go run` (skips build). Usage: `make run.go BIN=user-center`. |
| `deploy.infra` | Start infrastructure dependencies (MySQL, Redis). Usage: `make deploy.infra`. |
| `deploy.run` | Start all services using Docker Compose. |
| `deploy.down` | Stop and remove all Docker Compose services. |
| `run-example` | Run the example server. |
| `update-goagent` | Sync `goagent` code from upstream. |
| `update-logger` | Sync `logger` code from upstream. |

### Helper

| Target | Description |
| :--- | :--- |
| `help` | Display categorized help info. |
| `targets` | Show all targets grouped by source file. |

## Detailed Workflows

### Setting up the Environment

When starting, run `deps` to install necessary tools like `golangci-lint`, `mockgen`, and `protoc` plugins.

```bash
make deps
```

### Developing Code

1.  **Modify code**: Make your changes.
2.  **Format**: Run `make fmt` to ensure your code is formatted correctly and imports are organized.
    ```bash
    make fmt
    ```
3.  **Lint**: Run `make lint` to check for common errors.
    ```bash
    make lint
    ```
4.  **Test**: Run tests to ensure no regressions.
    ```bash
    make test
    ```
5.  **Run**: Run the service locally to verify behavior.
    ```bash
    make run-api
    ```

### Working with Protobuf

If you modify files in `pkg/api` (Protobuf definitions):

1.  Regenerate the code:
    ```bash
    make gen.proto
    ```
2.  If you need to clear old files:
    ```bash
    make clean.proto
    ```

### Building and Deploying Images

To build the docker image for the API server:

```bash
make image IMAGES="api"
```

To build and push for both AMD64 and ARM64:


```bash
make push.multiarch IMAGES="api"
```

### Advanced Targets

These targets are available in sub-makefiles but are generally used by the main targets or for specific granular tasks.

| Target | Description | Source |
| :--- | :--- | :--- |
| `go.build.<PLATFORM>.<BINARY>` | Build binary for specific platform. | `golang.mk` |
| `go.clean` | Clean build artifacts. | `golang.mk` |
| `go.test` | Run unit tests. | `golang.mk` |
| `go.test.cover` | Run unit tests with coverage. | `golang.mk` |
| `go.fmt` | Format source code. | `golang.mk` |
| `go.lint` | Run linters. | `golang.mk` |
| `tools.install` | Install all tools. | `tools.mk` |
| `tools.install.<TOOL>` | Install specific tool. | `tools.mk` |
| `tools.verify.<TOOL>` | Verify specific tool is installed. | `tools.mk` |
| `image.verify` | Verify docker version. | `image.mk` |
| `image.daemon.verify` | Verify docker daemon version. | `image.mk` |
| `image.dockerfile` | Generate all dockerfiles. | `image.mk` |
| `image.dockerfile.<IMAGE>` | Generate Dockerfile for specific image. | `image.mk` |
| `image.build.<PLATFORM>.<IMAGE>` | Build specified docker image. | `image.mk` |
| `image.push.<PLATFORM>.<IMAGE>` | Build and push specified docker image. | `image.mk` |
| `image.push.<PLATFORM>.<IMAGE>` | Build and push specified docker image. | `image.mk` |
| `run` | Run default server (api). Support `ENV`. | `run.mk` |
| `run.<BINARY>` | Run specified server. Auto-detects config. | `run.mk` |
| `run.go` | Run specified server directly (no build). | `run.mk` |
| `deploy.run` | Start all services. | `deploy.mk` |
| `deploy.down` | Stop all services. | `deploy.mk` |
| `deploy.infra` | Start infrastructure services only. | `deploy.mk` |
| `gen.proto` | Generate Proto codes. | `gen.mk` |
| `gen.clean` | Clean generated protobuf files. | `gen.mk` |
| `update` | Update vendor dependencies. | `update.mk` |
| `update-goagent` | Sync goagent from upstream. | `update.mk` |
| `update-logger` | Sync logger from upstream. | `update.mk` |

### Helper Scripts

The `scripts/` directory contains helper scripts that underpin the Makefile. While usually invoked by `make`, they can be run directly.

#### `scripts/image.sh`

Wrapper for image operations.

```bash
# Build images (default)
./scripts/image.sh build

# Push images
./scripts/image.sh push
```

#### `scripts/buildx.sh`

Wrapper for multi-architecture image operations.

```bash
# Build multi-arch images
./scripts/buildx.sh build

# Push multi-arch images
./scripts/buildx.sh push
```

#### `scripts/install/protobuf.sh`

Installs Protocol Buffers tools (buf, protoc-gen-go, etc.).

```bash
./scripts/install/protobuf.sh
```

#### `scripts/gen-dockerfile.sh`

Generates Dockerfiles for the project. Used by `make image.dockerfile`.

```bash
./scripts/gen-dockerfile.sh <OUTPUT_DIR> <IMAGE_NAME>
```

-   `scripts/make-rules/run.mk`: `run` commands.
-   `scripts/make-rules/update.mk`: `update` commands.
-   `scripts/make-rules/tools.mk`: Tools installation. Used by `make tools.install`.

#### `scripts/install/install.sh`

Core script for tool installation. Used by `make tools.install`.

```bash
./scripts/install/install.sh [TOOL_NAME]
```

#### `scripts/verify-sonic.sh`

Verifies the Sonic JSON integration. Used by `make verify-sonic`.

```bash
./scripts/verify-sonic.sh
```



